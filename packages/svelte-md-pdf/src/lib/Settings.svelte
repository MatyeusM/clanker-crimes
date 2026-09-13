<script>
  import { X, RotateCcw } from 'lucide-svelte';
  import { settings, resetSettings, PAPER_SIZES } from './pdf.js';
  import { FONTS } from './pdf.js';

  let { open = $bindable(false) } = $props();

  let dialog = $state();
  // Move keyboard focus into the modal when it opens (no autofocus attr).
  $effect(() => {
    if (open) dialog?.focus();
  });

  function close() {
    open = false;
  }

  function patch(obj) {
    settings.update((s) => ({ ...s, ...obj }));
  }

  function patchMargin(key, raw) {
    const v = Math.min(60, Math.max(5, Math.round(Number(raw) || 0)));
    settings.update((s) => ({ ...s, margins: { ...s.margins, [key]: v } }));
  }

  function onKey(e) {
    if (e.key === 'Escape') close();
  }
</script>

{#if open}
  <div
    class="overlay"
    onclick={(e) => {
      if (e.target === e.currentTarget) close();
    }}
    onkeydown={onKey}
    role="presentation"
  >
    <div
      class="modal"
      role="dialog"
      aria-modal="true"
      aria-label="PDF settings"
      bind:this={dialog}
      tabindex="-1"
    >
      <header>
        <h2>PDF settings</h2>
        <button type="button" class="icon" onclick={close} aria-label="Close settings">
          <X size={18} />
        </button>
      </header>

      <label>
        <span>Heading font</span>
        <select
          value={$settings.headingFont}
          onchange={(e) => patch({ headingFont: e.target.value })}
        >
          {#each FONTS as f}
            <option value={f.id}>{f.label}</option>
          {/each}
        </select>
      </label>

      <label>
        <span>Body font</span>
        <select
          value={$settings.bodyFont}
          onchange={(e) => patch({ bodyFont: e.target.value })}
        >
          {#each FONTS as f}
            <option value={f.id}>{f.label}</option>
          {/each}
        </select>
      </label>

      <label>
        <span>Paper size</span>
        <select
          value={$settings.paper}
          onchange={(e) => patch({ paper: e.target.value })}
        >
          {#each Object.entries(PAPER_SIZES) as [id, p]}
            <option value={id}>{p.label} ({p.width} × {p.height} mm)</option>
          {/each}
        </select>
      </label>

      <label class="check">
        <input
          type="checkbox"
          checked={$settings.pageNumbers}
          onchange={(e) => patch({ pageNumbers: e.currentTarget.checked })}
        />
        <span>Page numbers in the bottom margin</span>
      </label>

      <fieldset>
        <legend>Margins (mm)</legend>
        <div class="margins">
          {#each [['top', 'Top'], ['bottom', 'Bottom'], ['left', 'Left'], ['right', 'Right']] as [key, label]}
            <label>
              <span>{label}</span>
              <input
                type="number"
                min="5"
                max="60"
                step="1"
                value={$settings.margins[key]}
                oninput={(e) => patchMargin(key, e.target.value)}
              />
            </label>
          {/each}
        </div>
      </fieldset>

      <p class="note">Settings apply on the next Compile.</p>

      <footer>
        <button type="button" class="ghost" onclick={resetSettings}>
          <RotateCcw size={15} />
          Reset defaults
        </button>
        <button type="button" class="primary" onclick={close}>Done</button>
      </footer>
    </div>
  </div>
{/if}

<style>
  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 0.75rem;
  }
  h2 {
    margin: 0;
    font-size: 1.1rem;
  }
  .icon {
    display: inline-flex;
    padding: 0.25rem;
    border: 1px solid transparent;
    border-radius: 6px;
    background: none;
    cursor: pointer;
  }
  .icon:hover {
    background: #f0f0f0;
  }
  label {
    display: block;
    margin-bottom: 0.75rem;
  }
  label > span {
    display: block;
    font-size: 0.8rem;
    color: #555;
    margin-bottom: 0.25rem;
  }
  select,
  input[type='number'] {
    width: 100%;
    box-sizing: border-box;
    padding: 0.4rem 0.5rem;
    font-size: 0.9rem;
  }
  fieldset {
    border: 1px solid #e3e3e3;
    border-radius: 8px;
    margin: 0 0 0.75rem;
    padding: 0.75rem;
  }
  legend {
    font-size: 0.8rem;
    color: #555;
    padding: 0 0.25rem;
  }
  .check {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 0.75rem;
    cursor: pointer;
  }
  .check input {
    width: auto;
  }
  .check > span {
    margin: 0;
    font-size: 0.9rem;
    color: inherit;
  }
  .margins {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0 0.75rem;
  }
  .margins label {
    margin-bottom: 0.5rem;
  }
  .note {
    font-size: 0.8rem;
    color: #888;
    margin: 0 0 0.75rem;
  }
  footer {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  .ghost {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    background: none;
    border: 1px solid transparent;
    cursor: pointer;
    color: #555;
    padding: 0.4rem 0.5rem;
  }
  .ghost:hover {
    background: #f0f0f0;
  }
  .primary {
    padding: 0.45rem 1.25rem;
    cursor: pointer;
  }
</style>
