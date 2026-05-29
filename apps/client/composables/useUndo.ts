import { ref, onMounted, onUnmounted } from "vue";

export type UndoAction = {
  label: string;
  execute: () => Promise<void> | void;
};

const UNDO_WINDOW_MS = 6000;

const current = ref<UndoAction | null>(null);
let timer: ReturnType<typeof setTimeout> | null = null;

export function pushUndo(action: UndoAction) {
  if (timer) clearTimeout(timer);
  current.value = action;
  timer = setTimeout(() => {
    current.value = null;
    timer = null;
  }, UNDO_WINDOW_MS);
}

export async function undoLast() {
  if (!current.value) return;
  const action = current.value;
  dismissUndo();
  try {
    await action.execute();
  } catch (err) {
    console.error("Undo failed:", err);
  }
}

export function dismissUndo() {
  if (timer) clearTimeout(timer);
  current.value = null;
  timer = null;
}

export function useUndo() {
  function onKeydown(e: KeyboardEvent) {
    const mod = navigator.platform.includes("Mac") ? e.metaKey : e.ctrlKey;
    if (mod && e.key === "z" && !e.shiftKey && current.value) {
      e.preventDefault();
      undoLast();
    }
  }

  onMounted(() => window.addEventListener("keydown", onKeydown));
  onUnmounted(() => window.removeEventListener("keydown", onKeydown));

  return {
    current,
    pushUndo,
    undoLast,
    dismissUndo,
  };
}
