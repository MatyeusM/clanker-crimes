<script>
  import { get } from 'svelte/store';
  import { Settings as SettingsIcon, Download as DownloadIcon } from 'lucide-svelte';
  import Editor from './lib/Editor.svelte';
  import Preview from './lib/Preview.svelte';
  import SettingsModal from './lib/Settings.svelte';
  import { pdfBlob, compileStatus } from './lib/pdf.js';
  import { settings, paperLabel } from './lib/pdf.js';
  import { prefetchFonts } from './lib/pdf.js';
  import { SAMPLE_MARKDOWN } from './lib/pdf.js';

  let markdown = $state(SAMPLE_MARKDOWN);
  let settingsOpen = $state(false);

  // Warm the font cache while the browser is idle, so the first Compile
  // rarely waits on TTF downloads. Re-runs when the chosen fonts change;
  // the shared fetch cache makes repeats free.
  $effect(() => {
    const ids = [$settings.headingFont, $settings.bodyFont];
    if (typeof window === 'undefined') return;
    let cancel = () => {};
    const run = () => {
      prefetchFonts(...ids).catch(() => {});
    };
    if ('requestIdleCallback' in window) {
      const handle = window.requestIdleCallback(run, { timeout: 3000 });
      cancel = () => window.cancelIdleCallback(handle);
    } else {
      const timer = setTimeout(run, 1200);
      cancel = () => clearTimeout(timer);
    }
    return cancel;
  });

  // NOTE: the document model is computed inline here, on explicit user
  // action only — never in a reactive statement watching the textarea.
  // Heavy deps (markdown-it, jsPDF) load on first compile, keeping the
  // initial bundle small.
  async function handleCompile() {
    compileStatus.set({ state: 'compiling', message: 'Compiling…', pageCount: 0 });
    await new Promise((r) => setTimeout(r, 0)); // let the status paint first
    try {
      const s = get(settings);
      const { parseMarkdown, compileToPdf } = await import('./lib/pdf.js');
      const model = await parseMarkdown(markdown);
      const { doc, pageCount } = await compileToPdf(model, s);
      pdfBlob.set(doc.output('blob'));
      compileStatus.set({
        state: 'done',
        message: `Compiled ${pageCount} page(s) · ${paperLabel(s.paper)}${s.pageNumbers ? ' · numbered' : ''}.`,
        pageCount,
      });
    } catch (e) {
      compileStatus.set({
        state: 'error',
        message: 'Compile failed: ' + (e?.message ?? e),
        pageCount: 0,
      });
    }
  }

  function handleDownload() {
    const blob = get(pdfBlob);
    if (!blob) return;
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'document.pdf';
    a.click();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  }
</script>

<header>
  <h1>Markdown → PDF</h1>
  <div class="actions">
    <button type="button" class="with-icon" onclick={() => (settingsOpen = true)}>
      <SettingsIcon size={16} />
      Settings
    </button>
    <button type="button" onclick={handleCompile} disabled={$compileStatus.state === 'compiling'}>
      {$compileStatus.state === 'compiling' ? 'Compiling…' : 'Compile'}
    </button>
    <button
      type="button"
      class="with-icon"
      onclick={handleDownload}
      disabled={!$pdfBlob}
    >
      <DownloadIcon size={16} />
      PDF
    </button>
  </div>
  <p class="status" class:error={$compileStatus.state === 'error'}>
    {$compileStatus.message}
  </p>
</header>

<main>
  <Editor bind:markdown />
  <Preview />
</main>

<SettingsModal bind:open={settingsOpen} />

<style>
  header {
    display: flex;
    align-items: center;
    gap: 1rem;
    flex-wrap: wrap;
    padding: 0.75rem 1rem;
    border-bottom: 1px solid #ddd;
  }
  h1 {
    font-size: 1.2rem;
    margin: 0;
  }
  .actions {
    display: flex;
    gap: 0.5rem;
  }
  .with-icon {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
  }
  .status {
    margin: 0 0 0 auto;
    font-size: 0.85rem;
    color: #555;
  }
  .status.error {
    color: #b00020;
  }
  main {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1rem;
    padding: 1rem;
    height: calc(100vh - 65px);
    box-sizing: border-box;
  }
  @media (max-width: 800px) {
    main {
      grid-template-columns: 1fr;
      height: auto;
    }
  }
  button {
    cursor: pointer;
  }
  button:disabled {
    cursor: default;
    opacity: 0.5;
  }
</style>
