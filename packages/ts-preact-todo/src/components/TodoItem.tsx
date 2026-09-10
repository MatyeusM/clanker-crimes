import { Check, Pencil, Trash2, X } from "lucide-preact";
import { useState } from "preact/hooks";

import { TodoSchema } from "../lib/todos";

type TodoItemProps = {
  onDelete: any;
  onToggle: any;
  onUpdate: any;
  todo: any;
};

export function TodoItem({
  onDelete,
  onToggle,
  onUpdate,
  todo,
}: TodoItemProps) {
  const [editing, setEditing] = useState(false);
  const [draftTitle, setDraftTitle] = useState(todo.title);
  const [draftNotes, setDraftNotes] = useState(todo.notes ?? "");
  const [draftPriority, setDraftPriority] = useState<any>(todo.priority);
  const [error, setError] = useState<any>(null);

  function startEdit(): void {
    setDraftTitle(todo.title);
    setDraftNotes(todo.notes ?? "");
    setDraftPriority(todo.priority);
    setError(null);
    setEditing(true);
  }

  function cancelEdit(): void {
    setEditing(false);
    setError(null);
  }

  function saveEdit(event: any): void {
    event.preventDefault();
    const parsed = TodoSchema.pick({
      notes: true,
      priority: true,
      title: true,
    }).safeParse({
      notes: draftNotes,
      priority: draftPriority,
      title: draftTitle,
    });
    if (!parsed.success) {
      const firstIssue: any = parsed.error.issues[0];
      setError(firstIssue ? firstIssue.message : "Could not save changes");
      return;
    }
    const updateError: any = onUpdate(todo.id, parsed.data);
    if (updateError) {
      setError(updateError);
      return;
    }
    setEditing(false);
    setError(null);
  }

  if (editing) {
    return (
      <li class="todo todo--editing">
        <form
          class="edit__form"
          onSubmit={saveEdit}>
          <input
            class="input"
            maxLength={120}
            onInput={(event: any) => setDraftTitle(event.currentTarget.value)}
            value={draftTitle}
          />
          <input
            class="input"
            maxLength={500}
            onInput={(event: any) => setDraftNotes(event.currentTarget.value)}
            placeholder="Notes (optional)"
            value={draftNotes}
          />
          <div class="edit__row">
            <select
              class="input input--select"
              onChange={(event: any) =>
                setDraftPriority(event.currentTarget.value)
              }
              value={draftPriority}>
              <option value="low">Low</option>
              <option value="medium">Medium</option>
              <option value="high">High</option>
            </select>
            <div class="edit__actions">
              <button
                class="btn btn--ghost"
                onClick={cancelEdit}
                type="button">
                <X size={16} />
                Cancel
              </button>
              <button
                class="btn btn--primary"
                type="submit">
                <Check size={16} />
                Save
              </button>
            </div>
          </div>
          {error && <p class="form__error">{error}</p>}
        </form>
      </li>
    );
  }

  return (
    <li class={todo.completed ? "todo todo--done" : "todo"}>
      <button
        aria-label={
          todo.completed
            ? `Mark ${todo.title} as active`
            : `Mark ${todo.title} as done`
        }
        aria-pressed={todo.completed}
        class="todo__check"
        onClick={() => onToggle(todo.id)}
        type="button">
        {todo.completed && <Check size={16} />}
      </button>

      <div class="todo__body">
        <div class="todo__top">
          <p class="todo__title">{todo.title}</p>
          <span class={`pill pill--${todo.priority}`}>{todo.priority}</span>
        </div>
        {todo.notes && <p class="todo__notes">{todo.notes}</p>}
      </div>

      <div class="todo__actions">
        <button
          aria-label={`Edit ${todo.title}`}
          class="icon-btn"
          onClick={startEdit}
          type="button">
          <Pencil size={16} />
        </button>
        <button
          aria-label={`Delete ${todo.title}`}
          class="icon-btn icon-btn--danger"
          onClick={() => onDelete(todo.id)}
          type="button">
          <Trash2 size={16} />
        </button>
      </div>
    </li>
  );
}
