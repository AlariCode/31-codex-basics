// One request at a time prevents a slow response from replacing a newer observation.
export function startPolling(load: (signal: AbortSignal) => Promise<void>, interval: number): () => void {
  let controller: AbortController | null = null;
  let timer: ReturnType<typeof setTimeout> | undefined;
  let stopped = false;

  async function run() {
    if (stopped || document.hidden || controller) return;
    const request = new AbortController();
    controller = request;
    try {
      await load(request.signal);
    } finally {
      if (controller === request) {
        controller = null;
        if (!stopped && !document.hidden) timer = setTimeout(() => void run(), interval);
      }
    }
  }

  function visibilityChanged() {
    clearTimeout(timer);
    controller?.abort();
    controller = null;
    if (!document.hidden) void run();
  }

  document.addEventListener("visibilitychange", visibilityChanged);
  void run();
  return () => {
    stopped = true;
    clearTimeout(timer);
    controller?.abort();
    document.removeEventListener("visibilitychange", visibilityChanged);
  };
}
