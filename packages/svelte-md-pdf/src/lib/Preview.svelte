<script>
  import workerUrl from 'pdfjs-dist/build/pdf.worker.min.mjs?url';
  import { pdfBlob } from './pdf.js';

  // pdf.js loads on first preview, not with the initial bundle.
  let pdfjsPromise = null;
  function getPdfjs() {
    pdfjsPromise ??= import('pdfjs-dist/build/pdf.mjs').then((lib) => {
      lib.GlobalWorkerOptions.workerSrc = workerUrl;
      return lib;
    });
    return pdfjsPromise;
  }

  const DEFAULT_SCALE = 1.5;
  const MIN_SCALE = 0.5;
  const MAX_SCALE = 4;

  let canvas = $state();
  let wrap = $state();
  let pdfDoc = $state(null);
  let pageNum = $state(1);
  let pageCount = $state(0);
  let scale = $state(DEFAULT_SCALE);
  let loading = $state(false);
  let error = $state('');

  const zoomLabel = $derived(Math.round((scale / DEFAULT_SCALE) * 100) + '%');

  // React to each new compile via the shared blob store.
  $effect(() => {
    if ($pdfBlob) openBlob($pdfBlob);
  });

  // Ctrl/Cmd + wheel zooms the preview only. The listener must be
  // non-passive so the browser's own page zoom can be suppressed.
  // Plain wheel (no modifier) keeps its normal scroll behavior.
  $effect(() => {
    const el = wrap;
    if (!el) return;
    const onWheel = (e) => {
      if (!(e.ctrlKey || e.metaKey)) return;
      e.preventDefault();
      zoomBy(e.deltaY < 0 ? 1.15 : 1 / 1.15);
    };
    el.addEventListener('wheel', onWheel, { passive: false });
    return () => el.removeEventListener('wheel', onWheel);
  });

  async function openBlob(blob) {
    loading = true;
    error = '';
    try {
      const buf = await blob.arrayBuffer();
      try {
        await pdfDoc?.destroy();
      } catch {
        // ignore cleanup errors from a superseded document
      }
      const pdfjsLib = await getPdfjs();
      pdfDoc = await pdfjsLib.getDocument({ data: buf }).promise;
      pageCount = pdfDoc.numPages;
      pageNum = 1;
      scale = DEFAULT_SCALE;
      await renderPage(1);
    } catch (e) {
      error = 'Preview failed: ' + (e?.message ?? e);
    } finally {
      loading = false;
    }
  }

  async function renderPage(n, resetScroll = true) {
    if (!pdfDoc || !canvas) return;
    loading = true;
    try {
      const page = await pdfDoc.getPage(n);
      const dpr = window.devicePixelRatio || 1;
      const viewport = page.getViewport({ scale: scale * dpr });
      canvas.width = Math.floor(viewport.width);
      canvas.height = Math.floor(viewport.height);
      // Display at the readable CSS size; the backing store stays sharp.
      canvas.style.width = Math.floor(viewport.width / dpr) + 'px';
      canvas.style.height = Math.floor(viewport.height / dpr) + 'px';
      const ctx = canvas.getContext('2d');
      await page.render({ canvasContext: ctx, viewport }).promise;
      if (resetScroll) wrap?.scrollTo({ top: 0, left: 0 });
    } catch (e) {
      error = 'Render failed: ' + (e?.message ?? e);
    } finally {
      loading = false;
    }
  }

  function go(delta) {
    const next = Math.min(Math.max(1, pageNum + delta), pageCount);
    if (next !== pageNum) {
      pageNum = next;
      renderPage(next);
    }
  }

  function zoomBy(factor) {
    if (!pdfDoc || loading) return;
    scale = Math.min(Math.max(MIN_SCALE, scale * factor), MAX_SCALE);
    renderPage(pageNum, false);
  }
</script>

<div class="preview" role="region" aria-label="PDF preview">
  <div class="toolbar">
    <button
      type="button"
      onclick={() => go(-1)}
      disabled={!pdfDoc || loading || pageNum <= 1}
    >
      ← Prev
    </button>
    <span class="page">
      {#if pdfDoc}
        Page {pageNum} of {pageCount}
      {:else}
        No PDF yet
      {/if}
    </span>
    <button
      type="button"
      onclick={() => go(1)}
      disabled={!pdfDoc || loading || pageNum >= pageCount}
    >
      Next →
    </button>
    <span class="spacer"></span>
    {#if pdfDoc}
      <span class="zoom" title="Hold Ctrl and scroll to zoom">{zoomLabel}</span>
    {:else}
      <span class="hint">Ctrl + scroll to zoom</span>
    {/if}
    {#if loading && pdfDoc}
      <span class="loading">Loading…</span>
    {/if}
  </div>
  <div class="canvas-wrap" bind:this={wrap}>
    {#if !pdfDoc && !loading}
      {#if error}
        <p class="error">{error}</p>
      {:else}
        <p class="muted">
          Press <strong>Compile</strong> to render a preview here.<br />
          <span class="hint">Tip: hold Ctrl and scroll over the page to zoom.</span>
        </p>
      {/if}
    {:else if error}
      <p class="error">{error}</p>
    {:else}
      <div class="page-holder" hidden={!pdfDoc}>
        <canvas bind:this={canvas}></canvas>
      </div>
    {/if}
  </div>
</div>

<style>
  .preview {
    display: flex;
    flex-direction: column;
    min-height: 0;
    border: 1px solid #ddd;
    border-radius: 8px;
    overflow: hidden;
    background: #fff;
  }
  .spacer {
    flex: 1;
  }
  .zoom,
  .hint {
    font-size: 0.8rem;
    color: #888;
    white-space: nowrap;
  }
  .zoom {
    min-width: 3rem;
    text-align: right;
  }
  .canvas-wrap {
    flex: 1;
    overflow: auto;
    background: #525659;
    padding: 1rem;
    min-height: 320px;
    box-sizing: border-box;
  }
  /* Natural page size, centered when it fits, scrollable when it doesn't.
     width: max-content keeps margin-auto centering scroll-safe. */
  .page-holder {
    width: max-content;
    margin: 0 auto;
  }
  canvas {
    display: block;
    background: #fff;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.4);
  }
  .muted,
  .error {
    color: #ddd;
    text-align: center;
    margin-top: 3rem;
    line-height: 1.8;
  }
  .error {
    color: #ffb4b4;
  }
  button {
    cursor: pointer;
  }
  button:disabled {
    cursor: default;
    opacity: 0.5;
  }
</style>
