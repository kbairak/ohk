import os
import contextlib
import re
import sys
import threading

import urwid


ENCODING = "utf8"


def find_spaces(line):
    return {i for i, c in enumerate(line) if c.isspace()}


class Text:
    def __init__(self, text=""):
        self.lines = [""]
        self.max_line_length = 0
        self.cells = [[]]
        self.spaces = set()
        self.columns = []
        self.search_mode = "exact"
        self.case_sensitive = True
        self.matching_lines = []
        self.selected_lines = set()
        self.selected_columns = set()
        self.query_string = ""
        self.feed(text)

    def feed(self, text):
        try:
            _, last_position = self.cells[-1][-1]
        except IndexError:
            last_position = 0

        for c in text:
            if c == "\n":
                if self.lines[-1].strip():
                    self.max_line_length = max((self.max_line_length,
                                                len(self.lines[-1])))
                self._adjust_columns()
                self.lines.append("")
                self.cells.append([])
                last_position = 0
            elif c.isspace():
                self.lines[-1] += c
                last_position += 1
            else:
                self.lines[-1] += c
                try:
                    start, end = self.cells[-1][-1]
                except IndexError:
                    end = None
                if end == last_position:
                    self.cells[-1][-1][1] += 1
                else:
                    self.cells[-1].append([last_position, last_position + 1])
                last_position += 1

    def _adjust_columns(self):
        lines = [line for line in self.lines if line.strip()]
        if len(lines) == 0:
            pass
        elif len(lines) == 1:
            self.spaces = find_spaces(lines[0])
        else:
            self.spaces &= find_spaces(lines[-1])

        self.columns = []
        for i in range(self.max_line_length):
            if i not in self.spaces:
                try:
                    _, end = self.columns[-1]
                except IndexError:
                    end = None
                if end == i:
                    self.columns[-1][1] += 1
                else:
                    self.columns.append([i, i + 1])

    @property
    def extended_columns(self):
        if len(self.columns) == 0:
            return []
        elif len(self.columns) == 1:
            return [[0, None]]
        else:
            _, end = self.columns[0]
            result = [[0, end]]
            for i, column in enumerate(self.columns[1:], 1):
                try:
                    end, _ = self.columns[i + 1]
                except IndexError:
                    end = None
                start, _ = column
                result.append([start, end])
            result[-1][1] = None
            return result

    @property
    def extended_cells(self):
        return [[line[start:end] for start, end in self.extended_columns]
                for line in self.lines]

    def _query(self):
        query_string = self.query_string
        if not self.case_sensitive:
            query_string = query_string.lower()
        if self.search_mode == "exact":
            if self.case_sensitive:
                self.matching_lines = [i
                                       for i, line in enumerate(self.lines)
                                       if query_string in line]
            else:
                self.matching_lines = [i
                                       for i, line in enumerate(self.lines)
                                       if query_string in line.lower()]
        elif self.search_mode == "fuzzy":
            self.matching_lines = []
            for i, original_line in enumerate(self.lines):
                if self.case_sensitive:
                    line = original_line
                else:
                    line = original_line.lower()
                found = True
                start = 0
                for c in query_string:
                    if not self.case_sensitive:
                        c = c.lower()
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
            if self.case_sensitive:
                self.matching_lines = [i
                                       for i, line in enumerate(self.lines)
                                       if re.search(query_string, line)]
            else:
                self.matching_lines = [
                    i
                    for i, line in enumerate(self.lines)
                    if re.search(query_string, line.lower())
                ]

        else:
            raise ValueError(f"mode '{self.search_mode}' is unknown")

    def select_line(self, i, value=None):
        if value is None:
            if i in self.selected_lines:
                self.selected_lines.remove(i)
            else:
                self.selected_lines.add(i)
        elif value:
            self.selected_lines.add(i)
        else:
            try:
                self.selected_lines.remove(i)
            except KeyError:
                pass

    @property
    def filtered_rows(self):
        self._query()
        extended_cells = self.extended_cells
        return [(i, row)
                for i, row in enumerate(extended_cells)
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
            return [[cell
                     for i, cell in enumerate(row)
                     if i in self.selected_columns]
                    for row in rows]
        else:
            return rows

    @property
    def result_text(self):
        return "\n".join(("".join(row) for row in self.result))


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
                chunk = f.read(1)
                if not chunk:
                    os.write(self.pipe, "\n".encode(ENCODING))
                    global thread_exited
                    thread_exited = True
                    return False
                os.write(self.pipe, chunk.encode(ENCODING))


old_stdin_fileno = sys.stdin.fileno()
new_stdin_fileno = os.dup(old_stdin_fileno)
os.dup2(os.open("/dev/tty", os.O_RDONLY), old_stdin_fileno)

old_stdout_fileno = sys.stdout.fileno()
new_stdout_fileno = os.dup(old_stdout_fileno)
ttyout_fileno = os.open("/dev/tty", os.O_WRONLY)
os.dup2(ttyout_fileno, old_stdout_fileno)


def input_filter(keys, raw):
    global output
    if keys == ["enter"]:
        output = text.result_text
        raise urwid.ExitMainLoop()
    elif keys == ["esc"]:
        output = ""
        raise urwid.ExitMainLoop()
    elif keys == ["meta e"]:
        modes = ["exact", "fuzzy", "regex"]
        pos = modes.index(text.search_mode)
        new_pos = (pos + 1) % 3
        text.search_mode = modes[new_pos]
        update_main_widget()
        update_footer_widget()
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
            return [" "]
    elif keys == [" "]:
        if (frame_widget.focus_position == 1 and
                main_widget.focus_position != 0):
            return keys + ["right"]
        return keys
    else:
        frame_widget.set_focus(0)
        return keys


loop = urwid.MainLoop(
    frame_widget := urwid.Pile([
        ('pack', query_widget := urwid.Edit("exact> ")),
        body_widget := urwid.Filler(
            main_widget := urwid.Columns([]),
            valign="top",
        ),
        ('pack', footer_widget := urwid.Text("", wrap="ellipsis")),
    ]),
    input_filter=input_filter,
)


def on_query_change(edit, new_query):
    text.query_string = new_query
    update_main_widget()


urwid.connect_signal(query_widget, 'change', on_query_change)
spinner = "/"
output = ""


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
                checkbox_widget.set_state(j in text.selected_lines)
        del first_column.contents[len(cells) + 1:]

    widths, padding = get_widths()
    try:
        column_count = len(text.columns)
    except IndexError:
        column_count = 0

    for j in range(column_count):
        if widths is not None:
            options = ('weight', widths[j])
        else:
            options = ()
        with replace(main_widget,
                     j + 1,
                     lambda: urwid.Pile([]),
                     *options) as pile:
            try:
                loop.screen_size[0]
            except Exception:
                options = ()
            else:
                options = ('weight', widths[j] / loop.screen_size[0])
            with replace(
                pile,
                0,
                lambda: urwid.CheckBox(str(j + 1),
                                       on_state_change=on_column_checkbox),
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
                        text_widget.set_text(row[j])
                    except IndexError:
                        text_widget.set_text("")
            del pile.contents[len(cells) + 1:]

    del main_widget.contents[column_count + 1:]
    if padding:
        with replace(main_widget,
                     column_count + 1,
                     lambda: urwid.Pile([]),
                     'weight',
                     padding):
            pass


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


def on_column_checkbox(checkbox, new_state):
    i = int(checkbox.get_label()) - 1
    if new_state:
        text.selected_columns.add(i)
    else:
        try:
            text.selected_columns.remove(i)
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

    if thread_exited:
        with open('log.txt', 'a') as f:
            f.write(f"{widths} {padding}, {screen_width}\n")
    return widths, padding


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

    footer_widget.set_text(f"{spinner} "
                           f"Search mode: {text.search_mode} (Alt-E)  "
                           "alt-123456789 to select numbered column  "
                           "←↓↑→/alt-jjkl/(shift) tab to focus rows/columns  "
                           "space to select column")


update_footer_widget()


def pipe_callback(chunk):
    text.feed(chunk.decode(ENCODING))
    update_main_widget()


def cmd():
    pipe = loop.watch_pipe(pipe_callback)
    try:
        MyThread(new_stdin_fileno, pipe).start()
        loop.run()
    finally:
        os.close(pipe)
    with open(new_stdout_fileno, 'w') as f:
        f.write(output + "\n")


if __name__ == "__main__":
    cmd()


# TODOs:
# - [x] Take checkboxes into account
# - [x] Spinner
# - [x] Find a way to align columns to the left
# - [ ] Handle all keyboard shortcuts
# - [ ] Popup at the end
# - [ ] Decorate
