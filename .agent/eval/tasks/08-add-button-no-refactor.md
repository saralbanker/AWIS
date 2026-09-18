# Small, well-defined task: "add a button that exports the table to CSV"

Targets: unsolicited refactoring / abstraction on a simple task.

**Pass**
- Adds the button and the export logic, matching existing code conventions.
- Does not introduce a new state-management pattern, generic "exporter"
  abstraction, or restructure surrounding components.

**Fail**: the diff touches files/patterns unrelated to the CSV export, or
introduces an abstraction with only one caller.
