import argparse
import contextlib
import itertools
import os
import re
import shlex
import subprocess
import sys
import threading

import urwid


ENCODING = "utf8"


def find_spaces(line):
    return {i for i, c in enumerate(line) if c.isspace()}


class Text:
    def __init__(self, text="", search_mode="exact", case_sensitive=True):
        self.lines = [""]
        self.spaces = set()
        self.columns = []
        self.search_mode = search_mode
        self.case_sensitive = case_sensitive
        self.query_string = ""
        self.matching_lines = []
        self.selected_lines = set()
        self.selected_columns = set()

        self.feed(text)

    def feed(self, text):
        for c in text:
            if c == "\n":
                if self.lines[-1].strip():
                    self._adjust_columns()
                self.lines.append("")
            else:
                self.lines[-1] += c

    def _adjust_columns(self):
        lines = [line for line in self.lines if line.strip()]
        if len(lines) == 0:
            pass
        elif len(lines) == 1:
            self.spaces = find_spaces(lines[0])
        else:
            self.spaces &= find_spaces(lines[-1])

        self.columns = []
        # For all non-space positions
        for pos in sorted(set(range(max((len(line) for line in lines)))) -
                          self.spaces):
            try:
                _, end = self.columns[-1]
            except IndexError:
                end = None
            if end == pos:
                self.columns[-1][1] += 1
            else:
                self.columns.append([pos, pos + 1])

    def _query(self):
        query_string = self.query_string
        if not self.case_sensitive:
            query_string = query_string.lower()

        if self.search_mode == "exact":
            self.matching_lines = [
                i
                for i, line in enumerate(self.lines)
                if query_string in (line
                                    if self.case_sensitive
                                    else line.lower())
            ]

        elif self.search_mode == "fuzzy":
            self.matching_lines = []
            for i, line in enumerate(self.lines):
                if not self.case_sensitive:
                    line = line.lower()
                found = True
                start = 0
                for c in query_string:
                    try:
                        pos = line.index(c, start)
                    except ValueError:
                        found = False
                        break
                    else:
                        start = pos + 1
                if found:
                    self.matching_lines.append(i)

        elif self.search_mode == "regex":
            self.matching_lines = [
                i
                for i, line in enumerate(self.lines)
                if re.search(query_string,
                             line if self.case_sensitive else line.lower())
            ]

        else:
            raise ValueError(f"mode '{self.search_mode}' is unknown")

    def toggle_column(self, column_index):
        if column_index in self.selected_columns:
            self.selected_columns.remove(column_index)
        else:
            self.selected_columns.add(column_index)

    def toggle_line(self, line_index):
        if line_index in self.selected_lines:
            self.selected_lines.remove(line_index)
        else:
            self.selected_lines.add(line_index)

    @property
    def extended_columns(self):
        if len(self.columns) == 0:
            return []
        elif len(self.columns) == 1:
            return [[0, None]]
        else:
            result = [[column[0], self.columns[i + 1][0]]
                      for i, column in enumerate(self.columns[:-1])]
            result[0][0] = 0
            result.append([self.columns[-1][0], None])
            return result

    @property
    def extended_cells(self):
        return [[line[start:end] for start, end in self.extended_columns]
                for line in self.lines]

    @property
    def filtered_rows(self):
        self._query()
        return [(i, row)
                for i, row in enumerate(self.extended_cells)
                if i in self.matching_lines and any(row)]

    @property
    def result(self):
        self._query()
        extended_cells = self.extended_cells
        if self.selected_lines:
            rows = [row
                    for i, row in enumerate(extended_cells)
                    if i in self.selected_lines and i in self.matching_lines]
        else:
            rows = [row
                    for i, row in enumerate(extended_cells)
                    if i in self.matching_lines]

        if self.selected_columns:
            rows = [[cell
                     for i, cell in enumerate(row)
                     if i in self.selected_columns]
                    for row in rows]

        return "\n".join(("".join(row) for row in rows)) + "\n"


text = Text()


thread_exited = False


class MyThread(threading.Thread):
    daemon = True

    def __init__(self, stdin_fileno, pipe):
        self.stdin_fileno = stdin_fileno
        self.pipe = pipe
        super().__init__()

    def run(self, *args, **kwargs):
        with open(self.stdin_fileno) as f:
            while True:
                chunk = f.read(10)
                if not chunk:
                    os.write(self.pipe, "\n".encode(ENCODING))
                    global thread_exited
                    thread_exited = True
                    return False
                os.write(self.pipe, chunk.encode(ENCODING))


def input_filter(keys, raw):
    global output, text

    if help_visible:
        hide_help()
        return []

    if keys == ["enter"]:
        output = text.result
        raise urwid.ExitMainLoop()

    elif keys == ["esc"]:
        output = ""
        raise urwid.ExitMainLoop()

    elif keys == ["meta e"]:
        modes = ["exact", "fuzzy", "regex"]
        pos = modes.index(text.search_mode)
        new_pos = (pos + 1) % 3
        text.search_mode = modes[new_pos]
        update_query_widget()
        update_main_widget()

    elif keys == ["meta i"]:
        text.case_sensitive = not text.case_sensitive
        update_query_widget()
        update_main_widget()

    elif keys in (["left"], ["meta h"], ["shift tab"]):
        if frame_widget.focus_position == 0 and keys == ["left"]:
            return keys
        frame_widget.focus_position = 1
        new_pos = main_widget.focus_position - 1
        if new_pos == 0:
            new_pos = len(main_widget.contents) - 1
        main_widget.focus_position = new_pos
        main_widget.focus.focus_position = 0

    elif keys in (["right"], ["meta l"], ["tab"]):
        if frame_widget.focus_position == 0 and keys == ["right"]:
            return keys
        frame_widget.focus_position = 1
        new_pos = main_widget.focus_position + 1
        if new_pos >= len(main_widget.contents):
            new_pos = 1
        main_widget.focus_position = new_pos
        main_widget.focus.focus_position = 0

    elif keys in (["down"], ["meta j"]):
        if frame_widget.focus_position == 0 or main_widget.focus_position > 0:
            frame_widget.focus_position = 1
            main_widget.focus_position = 0
            main_widget.focus.focus_position = 1
        else:
            new_pos = main_widget.focus.focus_position + 1
            if new_pos >= len(main_widget.focus.contents):
                new_pos = 1
            main_widget.focus.focus_position = new_pos

    elif keys in (["up"], ["meta k"]):
        if frame_widget.focus_position == 0 or main_widget.focus_position > 0:
            frame_widget.focus_position = 1
            main_widget.focus_position = 0
            main_widget.focus.focus_position = \
                len(main_widget.focus.contents) - 1
        else:
            new_pos = main_widget.focus.focus_position - 1
            if new_pos == 0:
                new_pos = len(main_widget.focus.contents) - 1
            main_widget.focus.focus_position = new_pos

    elif len(keys) == 1 and keys[0] in (f"meta {i}" for i in range(1, 10)):
        new_pos = int(keys[0][-1])
        frame_widget.focus_position = 1
        try:
            main_widget.focus_position = new_pos
        except IndexError:
            pass
        else:
            text.toggle_column(new_pos - 1)
            return [" "]

    elif keys == ["meta a"]:
        if len(text.selected_lines) == len(text.lines):
            text.selected_lines.clear()
        else:
            text.selected_lines |= set(range(len(text.lines)))
        update_main_widget()

    elif keys == ["meta c"]:
        if len(text.selected_columns) == len(text.columns):
            text.selected_columns.clear()
        else:
            text.selected_columns |= set(range(len(text.columns)))
        update_main_widget()

    elif keys == [" "]:
        if (frame_widget.focus_position == 1 and
                main_widget.focus_position != 0):
            text.toggle_column(main_widget.focus_position - 1)
            return keys + ["right"]
        elif (frame_widget.focus_position == 1 and
              main_widget.focus_position == 0):
            text.toggle_line(main_widget.focus.focus_position - 1)
            return keys + ["down"]
        return keys

    elif keys == ["meta r"]:
        text = Text(text.result, text.search_mode, text.case_sensitive)
        query_widget.edit_text = ""
        update_query_widget()
        update_main_widget()

    elif keys == ["meta /"]:
        show_help()

    else:
        frame_widget.focus_position = 0
        return keys


loop = urwid.MainLoop(
    frame_widget := urwid.Pile([
        ('pack', query_widget := urwid.Edit("")),
        urwid.Filler(
            main_widget := urwid.Columns([]),
            valign="top",
        ),
        ('pack', footer_widget := urwid.Columns([], dividechars=2)),
    ]),
    input_filter=input_filter,
)


help_visible = False


def show_help():
    global help_visible
    if help_visible:
        return
    loop.widget = urwid.Overlay(
        urwid.LineBox(urwid.Filler(
            urwid.Padding(urwid.Text(
                "\n\n".join((
                    "Shortcuts:",
                    "- Esc: ",
                    "    Quit ohk",
                    "- Enter: ",
                    "    Finalize selection",
                    "- Alt-E: ",
                    "    Change search mode",
                    "- Alt-I: ",
                    "    Change case sensitivity",
                    "- ←↓↑→ / Alt-hjkl / (shift) tab: ",
                    "    Focus rows/columns",
                    "- Space: ",
                    "    Select row/column",
                    "- Alt-123456789: ",
                    "    Select numbered column",
                    "- Left click:",
                    "    Select column",
                    "- Alt-A/C: ",
                    "    Select all/none rows/columns",
                    "- Alt-R: ",
                    "    Rerun ohk with currently selected output",
                ))
            ), left=2, right=2),
            valign="top",
        )),
        frame_widget,
        align="center",
        width=('relative', 60),
        valign="middle",
        height=('relative', 70),
    )
    help_visible = True


def hide_help():
    global help_visible
    if not help_visible:
        return
    loop.widget = frame_widget
    help_visible = False


def update_query_widget():
    caption = [text.search_mode]
    if not text.case_sensitive:
        caption.append(" (case-ins)")
    caption.append("> ")
    query_widget.set_caption("".join(caption))


def on_query_change(edit, new_query):
    text.query_string = new_query
    update_main_widget()


urwid.connect_signal(query_widget, 'change', on_query_change)
spinner = "/"
output = ""


@contextlib.contextmanager
def replace(container, index, default, *options):
    try:
        widget, _ = container.contents[index]
    except IndexError:
        widget = default()

    yield widget

    try:
        container.contents[index] = (widget, container.options(*options))
    except IndexError:
        container.contents.append((widget, container.options(*options)))


def on_row_checkbox(checkbox, new_state):
    i = checkbox.user_data
    if new_state:
        text.selected_lines.add(i)
    else:
        try:
            text.selected_lines.remove(i)
        except KeyError:
            pass


def get_widths():
    try:
        screen_width = loop.screen_size[0]
    except Exception:
        return None, None

    widths = [end - start + 3 for start, end in text.columns]

    text_width = sum(widths)
    if text_width > screen_width - 4:
        padding = 0
        widths = [int(w * ((screen_width - 4) / text_width)) for w in widths]
    else:
        padding = screen_width - text_width - 4

    return widths, padding


class MouseSelectPile(urwid.Pile):
    def __init__(self, *args, column_index=None, **kwargs):
        self.column_index = column_index
        super().__init__(*args, **kwargs)

    def mouse_event(self, size, event, button, col, row, focus):
        if event != "mouse release":
            return super().mouse_event(size, event, button, col, row, focus)
        if self.column_index in text.selected_columns:
            text.selected_columns.remove(self.column_index)
        else:
            text.selected_columns.add(self.column_index)
        update_main_widget()


def update_main_widget():
    cells = text.filtered_rows

    with replace(main_widget,
                 0,
                 lambda: urwid.Pile([urwid.Text("")]),
                 'weight', 4) as first_column:
        for i, (j, _) in enumerate(cells):
            with replace(
                first_column,
                i + 1,
                lambda: urwid.CheckBox("", on_state_change=on_row_checkbox),
            ) as checkbox_widget:
                checkbox_widget.user_data = j
                checkbox_widget.state = j in text.selected_lines
        del first_column.contents[len(cells) + 1:]

    widths, padding = get_widths()

    for j in range(len(text.columns)):
        if widths is not None:
            options = ('weight', widths[j])
        else:
            options = ()
        with replace(main_widget,
                     j + 1,
                     lambda: MouseSelectPile([], column_index=j),
                     *options) as pile:
            try:
                loop.screen_size[0]
            except Exception:
                options = ()
            else:
                options = ('weight', widths[j])
            with replace(
                pile,
                0,
                lambda: urwid.CheckBox(str(j + 1)),
                *options
            ) as checkbox_widget:
                checkbox_widget.set_state(j in text.selected_columns,
                                          do_callback=False)

            for i, (_, row) in enumerate(cells):
                with replace(
                    pile,
                    i + 1,
                    lambda: urwid.Text("", wrap="ellipsis"),
                ) as text_widget:
                    try:
                        if text_widget.get_text()[0] != row[j]:
                            text_widget.set_text(row[j])
                    except IndexError:
                        if text_widget.get_text()[0] != "":
                            text_widget.set_text("")
            del pile.contents[len(cells) + 1:]

    del main_widget.contents[len(text.columns) + 1:]
    if padding:
        with replace(main_widget,
                     len(text.columns) + 1,
                     lambda: urwid.Pile([]),
                     'weight',
                     padding):
            pass


update_main_widget()


def update_footer_widget(_=None, user_data=None):
    global spinner
    if thread_exited:
        spinner = " "
    else:
        spinner_symbols = "/-\\|"
        pos = (spinner_symbols.index(spinner) + 1) % 4
        spinner = spinner_symbols[pos]
        try:
            loop.set_alarm_in(.05, update_footer_widget)
        except NameError:
            pass

    c = itertools.count()
    with replace(footer_widget,
                 next(c),
                 lambda: urwid.Text(""),
                 'given', 2) as widget:
        widget.set_text(spinner)

    with replace(footer_widget,
                 next(c),
                 lambda: urwid.Text("Alt-/ for help",
                                    align="right")) as widget:
        pass


update_footer_widget()


def pipe_callback(chunk):
    text.feed(chunk.decode(ENCODING))
    update_main_widget()


def read_command(msg, in_fd, out_fd):
    os.write(out_fd, msg.encode(ENCODING))
    command = []
    while True:
        c = os.read(in_fd, 1).decode(ENCODING)
        if c == "\n":
            break
        command.append(c)
    return "".join(command)


parser = argparse.ArgumentParser()
parser.add_argument("-f", "--fuzzy", action="store_true")
parser.add_argument("-r", "--regex", action="store_true")
parser.add_argument("-i", "--case-insensitive", action="store_true")


def cmd():
    keyboard_input = sys.stdin.fileno()
    pipe_input = os.dup(keyboard_input)
    os.dup2(os.open("/dev/tty", os.O_RDONLY), keyboard_input)

    old_stdout_fileno = sys.stdout.fileno()
    pipe_output = os.dup(old_stdout_fileno)
    tty_output = os.open("/dev/tty", os.O_WRONLY)
    os.dup2(tty_output, old_stdout_fileno)

    args = parser.parse_args()
    if args.fuzzy:
        text.search_mode = "fuzzy"
    elif args.regex:
        text.search_mode = "regex"
    text.case_sensitive = not args.case_insensitive

    update_query_widget()

    if os.isatty(pipe_input):
        command = read_command("Enter command: ", pipe_input, tty_output)
        process = subprocess.Popen(shlex.split(command),
                                   stdout=subprocess.PIPE,
                                   text=True,
                                   encoding=ENCODING)
        pipe_input = process.stdout.fileno()

    pipe = loop.watch_pipe(pipe_callback)
    try:
        MyThread(pipe_input, pipe).start()
        loop.run()
    finally:
        os.close(pipe)
    if os.isatty(pipe_output):
        command = read_command(
            "\n".join((
                "Tips:",
                "  - use 'xargs' to pass output as argument",
                "  - use 'cat' to print to terminal",
                "  - use 'ohk' to re-run ohk on the results",
                "Pipe output to: ",
            )),
            keyboard_input,
            tty_output,
        )
        process = subprocess.Popen(shlex.split(command),
                                   stdin=subprocess.PIPE,
                                   stdout=tty_output,
                                   stderr=tty_output,
                                   text=True,
                                   encoding=ENCODING)
        process.stdin.write(output)
        process.stdin.flush()
        process.stdin.close()
        process.wait()
    else:
        with open(pipe_output, 'w') as f:
            f.write(output + "\n")


if __name__ == "__main__":
    cmd()


# TODOs:
# - [x] Take checkboxes into account
# - [x] Spinner
# - [x] Find a way to align columns to the left
# - [x] Handle all keyboard shortcuts
# - [x] Ask follow-up command at the end
# - [x] Add case-insensitive trigger
# - [x] Command line options (-i etc)
# - [x] argparse
# - [x] Shortcuts for select all/none rows/columns
# - [x] Incrementally run ohk again (for compex searches etc)
# - [x] Help popup
# - [x] Select columns with mouse
# - [x] Speed things up
# - [ ] Decorate
# - [ ] When asking for following command, offer to open ohk again to its
#       output
# - [ ] Organize code better
# - [ ] Set output as environment variable on the outer shell
# - [ ] Use Ctrl- shortcuts
