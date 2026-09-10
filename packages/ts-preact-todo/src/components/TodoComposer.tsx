import { Plus } from "lucide-preact";
import { useState } from "preact/hooks";

import { NewTodoSchema } from "../lib/todos";

type ComposerProps = {
  onAdd: any;
};

const PRIORITIES: any = ["low", "medium", "high"];

export function TodoComposer({ onAdd }: ComposerProps) {
  const [title, setTitle] = useState("");
  const [notes, setNotes] = useState("");
  const [priority, setPriority] = useState<any>("medium");
  const [error, setError] = useState<any>(null);

  function handleSubmit(event: any): void {
    event.preventDefault();
    const parsed = NewTodoSchema.safeParse({ notes, priority, title });
    if (!parsed.success) {
      const firstIssue: any = parsed.error.issues[0];
      setError(firstIssue ? firstIssue.message : "Could not add this task");
      return;
    }
    const submitError: any = onAdd(parsed.data);
    if (submitError) {
      setError(submitError);
      return;
    }
    setTitle("");
    setNotes("");
    setPriority("medium");
    setError(null);
  }

  return (
    <form
      class="composer"
      onSubmit={handleSubmit}>
      <label class="field">
        <span class="field__label">New task</span>
        <input
          class={error ? "input input--error" : "input"}
          maxLength={120}
          name="title"
          onInput={(event: any) => setTitle(event.currentTarget.value)}
          placeholder="What needs doing?"
          value={title}
        />
      </label>

      <div class="composer__row">
        <label class="field field--inline">
          <span class="field__label">Priority</span>
          <div class="seg__group">
            {PRIORITIES.map((option: any) => (
              <button
                class={
                  option === priority
                    ? `seg seg--active seg--${option}`
                    : `seg seg--${option}`
                }
                key={option}
                onClick={(event: any) => {
                  event.preventDefault();
                  setPriority(option);
                }}
                type="button">
                {option}
              </button>
            ))}
          </div>
        </label>
        <button
          class="btn btn--primary"
          type="submit">
          <Plus size={18} />
          Add task
        </button>
      </div>

      <label class="field">
        <span class="field__label">Notes (optional)</span>
        <input
          class="input"
          maxLength={500}
          name="notes"
          onInput={(event: any) => setNotes(event.currentTarget.value)}
          placeholder="Details, links, context…"
          value={notes}
        />
      </label>

      {error && <p class="form__error">{error}</p>}
    </form>
  );
}
