import { nanoid } from "nanoid";
import { z } from "zod";

export const PrioritySchema = z.enum(["low", "medium", "high"]);
export type Priority = z.infer<typeof PrioritySchema>;

export const TodoSchema = z.object({
  completed: z.boolean(),
  createdAt: z.number().int(),
  id: z.string().min(1),
  notes: z.string().max(500).optional().default(""),
  priority: PrioritySchema.default("medium"),
  title: z.string().trim().min(1, "Give your task a title").max(120),
});

export type Todo = z.infer<typeof TodoSchema>;

export const TodoListSchema = z.array(TodoSchema);
export type TodoList = z.infer<typeof TodoListSchema>;

export const NewTodoSchema = z.object({
  notes: z.string().trim().max(500).default(""),
  priority: PrioritySchema.default("medium"),
  title: z.string().trim().min(1, "Give your task a title").max(120),
});

export type NewTodoInput = z.infer<typeof NewTodoSchema>;

export type TodoFilter = "active" | "all" | "completed";

const STORAGE_KEY = "clanker-todo:v1";
const SEED_PREFIX = "seed-";

function isBrowser(): boolean {
  return typeof window !== "undefined" && typeof localStorage !== "undefined";
}

export function isSeedId(id: any): boolean {
  return (id as string).startsWith(SEED_PREFIX);
}

export function loadTodos(fallback: any): any {
  if (!isBrowser()) {
    return fallback;
  }
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) {
      return fallback;
    }
    const parsed: any = JSON.parse(raw);
    const result = TodoListSchema.safeParse(parsed);
    if (!result.success) {
      return fallback;
    }
    return result.data;
  } catch {
    return fallback;
  }
}

export function saveTodos(todos: any): void {
  if (!isBrowser()) {
    return;
  }
  try {
    const validated = TodoListSchema.parse(todos);
    localStorage.setItem(STORAGE_KEY, JSON.stringify(validated));
  } catch {
    return;
  }
}

export function restoreSeeds(current: any, seeds: any): any {
  const list: any[] = Array.isArray(current) ? current : [];
  const userTodos = list.filter((todo: any) => !isSeedId(todo?.id ?? ""));
  return [...seeds, ...userTodos];
}

export function createTodoId(): string {
  return nanoid();
}

export function createTodo(input: any): any {
  const parsed = NewTodoSchema.parse(input);
  return {
    completed: false,
    createdAt: Date.now(),
    id: createTodoId(),
    notes: parsed.notes,
    priority: parsed.priority,
    title: parsed.title,
  };
}

export function partitionTodos(todos: any): any {
  const list: any[] = Array.isArray(todos) ? todos : [];
  return {
    active: list.filter((todo: any) => !todo?.completed),
    completed: list.filter((todo: any) => Boolean(todo?.completed)),
  };
}

export function summarizeTodos(todos: any): any {
  const list: any[] = Array.isArray(todos) ? todos : [];
  const byPriority: any = { high: 0, low: 0, medium: 0 };
  for (const todo of list) {
    const key = (todo as any)?.priority;
    if (key in byPriority) {
      byPriority[key] += 1;
    }
  }
  return { byPriority, total: list.length };
}

export function cloneTodoList(todos: any): any {
  return JSON.parse(JSON.stringify(todos ?? []));
}
