import re


def _find_spaces(line):
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
            self.spaces = _find_spaces(lines[0])
        else:
            self.spaces &= _find_spaces(lines[-1])

        self.columns = []
        # For all non-space positions
        for pos in sorted(set(range(max((len(line) for line in lines)))) -
                          self.spaces):
            try:
                # ***  ***  ***
                # [ ]  []
                #       ^
                _, end = self.columns[-1]
            except IndexError:
                # No columns yet
                end = None
            if end == pos:
                # ***  ***  ***
                # [ ]  [ ]
                #       -^
                self.columns[-1][1] += 1
            else:
                # ***  ***  ***
                # [ ]  [ ]  []
                #           ^
                self.columns.append([pos, pos + 1])

    def _query(self):
        if self.case_sensitive:
            query_string = self.query_string
            lines = self.lines
        else:
            query_string = query_string.lower()
            lines = [line.lower() for line in self.lines]

        if self.search_mode == "exact":
            self.matching_lines = [i
                                   for i, line in enumerate(lines)
                                   if query_string in line]

        elif self.search_mode == "fuzzy":
            self.matching_lines = []
            for i, line in enumerate(lines):
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
            try:
                pattern = re.compile(query_string)
            except Exception:
                return
            self.matching_lines = [i
                                   for i, line in enumerate(lines)
                                   if pattern.search(line)]

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
        """ Return widest possible columns based on the actual columns.

            line 1:           >    ***  **  **     ***   <
            line 2:           >    **   ******   *****   <
            columns:          >    [ ]  [    ]   [   ]   <
            extended_columns: > [      ][       ][     ] <
        """

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
