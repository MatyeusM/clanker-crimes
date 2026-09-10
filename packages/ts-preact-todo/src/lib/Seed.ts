const now = Date.now();
const hour = 60 * 60 * 1000;
const day = 24 * hour;

const SEEDS: any = [
  {
    completed: false,
    createdAgo: 2 * hour,
    id: "seed-01",
    notes: "oxlint + oxfmt must be clean.",
    priority: "high",
    title: "Ship Clanker TODO v1",
  },
  {
    completed: false,
    createdAgo: 5 * hour,
    id: "seed-02",
    notes: "Hero, features, pricing. Keep it focused on one action.",
    priority: "high",
    title: "Design the landing page",
  },
  {
    completed: false,
    createdAgo: 9 * hour,
    id: "seed-03",
    notes: "Filters, sorting, and empty states.",
    priority: "medium",
    title: "Build the todo list UI in Preact",
  },
  {
    completed: false,
    createdAgo: 14 * hour,
    id: "seed-04",
    notes: "Validate stored data before rendering.",
    priority: "medium",
    title: "Persist todos to localStorage",
  },
  {
    completed: true,
    createdAgo: 1 * day,
    id: "seed-05",
    notes: "",
    priority: "low",
    title: "Scaffold the project with Vite",
  },
  {
    completed: false,
    createdAgo: 1 * day + 3 * hour,
    id: "seed-06",
    notes: "Cover add, toggle, and filter flows.",
    priority: "high",
    title: "Write onboarding checklist",
  },
  {
    completed: false,
    createdAgo: 1 * day + 7 * hour,
    id: "seed-07",
    notes: "Group by inbox, today, and someday.",
    priority: "medium",
    title: "Triage the inbox",
  },
  {
    completed: true,
    createdAgo: 2 * day,
    id: "seed-08",
    notes: "Inter from Google Fonts, system fallback offline.",
    priority: "low",
    title: "Set up typography",
  },
  {
    completed: false,
    createdAgo: 2 * day + 4 * hour,
    id: "seed-09",
    notes: "Light default, dark via media query.",
    priority: "medium",
    title: "Polish the color theme",
  },
  {
    completed: false,
    createdAgo: 2 * day + 10 * hour,
    id: "seed-10",
    notes: "Focus ring must stay visible on every control.",
    priority: "high",
    title: "Audit keyboard navigation",
  },
  {
    completed: false,
    createdAgo: 3 * day,
    id: "seed-11",
    notes: "Plus, check, pencil, trash, search.",
    priority: "low",
    title: "Review icon usage",
  },
  {
    completed: true,
    createdAgo: 3 * day + 6 * hour,
    id: "seed-12",
    notes: "",
    priority: "medium",
    title: "Add the app header",
  },
  {
    completed: false,
    createdAgo: 3 * day + 12 * hour,
    id: "seed-13",
    notes: "Draft three options, pick one direction.",
    priority: "medium",
    title: "Sketch the empty state",
  },
  {
    completed: false,
    createdAgo: 4 * day,
    id: "seed-14",
    notes: "Keep titles under 120 characters.",
    priority: "low",
    title: "Tighten task copy",
  },
  {
    completed: false,
    createdAgo: 4 * day + 5 * hour,
    id: "seed-15",
    notes: "High, medium, low with clear colors.",
    priority: "medium",
    title: "Define priority labels",
  },
  {
    completed: true,
    createdAgo: 4 * day + 11 * hour,
    id: "seed-16",
    notes: "",
    priority: "low",
    title: "Configure Vite build",
  },
  {
    completed: false,
    createdAgo: 5 * day,
    id: "seed-17",
    notes: "Search across titles and notes.",
    priority: "medium",
    title: "Add task search",
  },
  {
    completed: false,
    createdAgo: 5 * day + 8 * hour,
    id: "seed-18",
    notes: "All, active, completed with counts.",
    priority: "medium",
    title: "Build filter tabs",
  },
  {
    completed: false,
    createdAgo: 6 * day,
    id: "seed-19",
    notes: "Show done ratio at a glance.",
    priority: "low",
    title: "Add progress bar",
  },
  {
    completed: false,
    createdAgo: 6 * day + 6 * hour,
    id: "seed-20",
    notes: "Confirm before destructive actions.",
    priority: "high",
    title: "Plan clear-completed flow",
  },
  {
    completed: true,
    createdAgo: 7 * day,
    id: "seed-21",
    notes: "",
    priority: "low",
    title: "Set up project README",
  },
  {
    completed: false,
    createdAgo: 7 * day + 4 * hour,
    id: "seed-22",
    notes: "Seven visible rows, then scroll.",
    priority: "medium",
    title: "Make the list scrollable",
  },
  {
    completed: false,
    createdAgo: 8 * day,
    id: "seed-23",
    notes: "Hover lift on add, pop on filter switch.",
    priority: "low",
    title: "Add micro-animations",
  },
  {
    completed: false,
    createdAgo: 8 * day + 9 * hour,
    id: "seed-24",
    notes: "Unique ids for every task.",
    priority: "medium",
    title: "Switch ids to nanoid",
  },
  {
    completed: false,
    createdAgo: 9 * day,
    id: "seed-25",
    notes: "Honor reduced-motion preferences.",
    priority: "low",
    title: "Check motion preferences",
  },
  {
    completed: false,
    createdAgo: 9 * day + 7 * hour,
    id: "seed-26",
    notes: "Invite two people for first impressions.",
    priority: "high",
    title: "Run a usability pass",
  },
  {
    completed: true,
    createdAgo: 10 * day,
    id: "seed-27",
    notes: "",
    priority: "medium",
    title: "Review zod schemas",
  },
  {
    completed: false,
    createdAgo: 10 * day + 6 * hour,
    id: "seed-28",
    notes: "Cold start should feel instant.",
    priority: "high",
    title: "Measure load performance",
  },
  {
    completed: false,
    createdAgo: 11 * day,
    id: "seed-29",
    notes: "Small screens first, then expand.",
    priority: "medium",
    title: "Test the mobile layout",
  },
  {
    completed: false,
    createdAgo: 11 * day + 8 * hour,
    id: "seed-30",
    notes: "Record a short demo for the portfolio.",
    priority: "low",
    title: "Prepare launch notes",
  },
];

export const seedTodos: any = SEEDS.map((seed: any) => ({
  completed: seed.completed,
  createdAt: now - seed.createdAgo,
  id: seed.id,
  notes: seed.notes,
  priority: seed.priority,
  title: seed.title,
}));

export function getSeedTitles(seeds: any = SEEDS): string[] {
  return (seeds as any[]).map((seed: any) => String(seed?.title ?? ""));
}

export function countSeedByPriority(seeds: any = SEEDS): any {
  const counts: any = { high: 0, low: 0, medium: 0 };
  for (const seed of seeds as any[]) {
    const key = (seed as any)?.priority;
    if (key in counts) {
      counts[key] += 1;
    }
  }
  return counts;
}
