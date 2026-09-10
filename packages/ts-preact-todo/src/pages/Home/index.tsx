import {
  ListTodo,
  PartyPopper,
  RotateCcw,
  Search,
  Sparkles,
  Trash2,
} from "lucide-preact";
import { useEffect, useMemo, useState } from "preact/hooks";

import { TodoComposer } from "../../components/TodoComposer.jsx";
import { TodoItem } from "../../components/TodoItem.jsx";
import { seedTodos } from "../../lib/Seed";
import {
  createTodo,
  loadTodos,
  restoreSeeds,
  saveTodos,
} from "../../lib/todos";

import "./style.css";

const FILTERS: any = ["all", "active", "completed"];

function sortTodos(todos: any): any {
  return [...todos].sort((a: any, b: any) => {
    if (a.completed !== b.completed) {
      return a.completed ? 1 : -1;
    }
    const weight: any = { high: 0, low: 2, medium: 1 };
    if (weight[a.priority] !== weight[b.priority]) {
      return weight[a.priority] - weight[b.priority];
    }
    return b.createdAt - a.createdAt;
  });
}

export function Home() {
  const [todos, setTodos] = useState<any>(() => loadTodos(seedTodos));
  const [filter, setFilter] = useState<any>("all");
  const [query, setQuery] = useState("");
  const [hydrated, setHydrated] = useState(false);

  useEffect(() => {
    setHydrated(true);
  }, []);

  useEffect(() => {
    if (hydrated) {
      saveTodos(todos);
    }
  }, [hydrated, todos]);

  const counts: any = useMemo(() => {
    const total = todos.length;
    const done = todos.filter((todo: any) => todo.completed).length;
    return { active: total - done, done, total };
  }, [todos]);

  const visible: any = useMemo(() => {
    const needle = query.trim().toLowerCase();
    const filtered = todos.filter((todo: any) => {
      if (filter === "active" && todo.completed) {
        return false;
      }
      if (filter === "completed" && !todo.completed) {
        return false;
      }
      if (!needle) {
        return true;
      }
      const haystack = `${todo.title} ${todo.notes ?? ""}`.toLowerCase();
      return haystack.includes(needle);
    });
    return sortTodos(filtered);
  }, [filter, query, todos]);

  const progress =
    counts.total === 0 ? 0 : Math.round((counts.done / counts.total) * 100);

  function addTodo(input: any): any {
    try {
      setTodos((prev: any) => [createTodo(input), ...prev]);
      return null;
    } catch {
      return "Could not add this task";
    }
  }

  function toggleTodo(id: any): void {
    setTodos((prev: any) =>
      prev.map((todo: any) =>
        todo.id === id ? { ...todo, completed: !todo.completed } : todo
      )
    );
  }

  function deleteTodo(id: any): void {
    setTodos((prev: any) => prev.filter((todo: any) => todo.id !== id));
  }

  function updateTodo(id: any, patch: any): any {
    try {
      setTodos((prev: any) =>
        prev.map((todo: any) => (todo.id === id ? { ...todo, ...patch } : todo))
      );
      return null;
    } catch {
      return "Could not save changes";
    }
  }

  function clearCompleted(): void {
    setTodos((prev: any) => prev.filter((todo: any) => !todo.completed));
  }

  function resetToSeed(): void {
    setTodos((prev: any) => restoreSeeds(prev, seedTodos));
    setFilter("all");
    setQuery("");
  }

  return (
    <div class="page">
      <section class="hero">
        <p class="hero__eyebrow">
          <Sparkles size={14} />
          Clanker TODO
        </p>
        <h1 class="hero__title">Get things done, one task at a time.</h1>
        <p class="hero__lede">
          A fast local-first todo list. Add tasks, set priorities, and pick them
          off. Everything stays in your browser.
        </p>

        <div
          class="progress__wrap"
          role="progressbar"
          aria-valuemax={100}
          aria-valuemin={0}
          aria-valuenow={progress}>
          <div class="progress__meta">
            <span>
              {counts.done} of {counts.total} done
            </span>
            <span>{progress}%</span>
          </div>
          <div class="progress">
            <div
              class="progress__bar"
              style={{ width: `${progress}%` }}
            />
          </div>
        </div>
      </section>

      <section class="card">
        <TodoComposer onAdd={addTodo} />

        <div class="card__toolbar">
          <div
            class="tabs"
            role="tablist"
            aria-label="Filter tasks">
            {FILTERS.map((option: any) => (
              <button
                class={
                  option === filter
                    ? "tabs__tab tabs__tab--active"
                    : "tabs__tab"
                }
                key={option}
                onClick={() => setFilter(option)}
                role="tab"
                aria-selected={option === filter}
                type="button">
                {option}
                <span class="tabs__count">
                  {option === "all"
                    ? counts.total
                    : option === "active"
                      ? counts.active
                      : counts.done}
                </span>
              </button>
            ))}
          </div>

          <label class="search">
            <Search size={16} />
            <input
              class="search__input"
              onInput={(event: any) => setQuery(event.currentTarget.value)}
              placeholder="Search tasks…"
              value={query}
            />
          </label>
        </div>

        {visible.length === 0 ? (
          <div class="empty">
            {counts.total === 0 ? (
              <PartyPopper size={28} />
            ) : (
              <ListTodo size={28} />
            )}
            <h2 class="empty__title">
              {counts.total === 0 ? "All clear" : "No tasks match"}
            </h2>
            <p class="empty__text">
              {counts.total === 0
                ? "Add your first task above to start your streak."
                : "Try a different search or filter."}
            </p>
            {counts.total > 0 && query.trim() !== "" && (
              <button
                class="btn btn--ghost"
                onClick={() => setQuery("")}
                type="button">
                Clear search
              </button>
            )}
          </div>
        ) : (
          <ul
            class="todo-list"
            key={`${filter}-${query.trim()}`}>
            {visible.map((todo: any) => (
              <TodoItem
                key={todo.id}
                onDelete={deleteTodo}
                onToggle={toggleTodo}
                onUpdate={updateTodo}
                todo={todo}
              />
            ))}
          </ul>
        )}

        <footer class="card__footer">
          <span class="card__meta">
            {counts.active} left · {counts.done} completed
          </span>
          <div class="card__footer-actions">
            <button
              class="btn btn--ghost"
              disabled={counts.done === 0}
              onClick={clearCompleted}
              type="button">
              <Trash2 size={16} />
              Clear completed
            </button>
            <button
              class="btn btn--ghost"
              onClick={resetToSeed}
              type="button">
              <RotateCcw size={16} />
              Reset demo
            </button>
          </div>
        </footer>
      </section>
    </div>
  );
}
