<script>
  let { markdown = $bindable('') } = $props();

  let fileInput = $state();
  let dragOver = $state(false);

  const lineCount = $derived(markdown ? markdown.split('\n').length : 0);
  const charCount = $derived(markdown?.length ?? 0);

  function pickFile() {
    fileInput?.click();
  }

  function onFileChosen(e) {
    loadFile(e.target.files?.[0]);
    e.target.value = '';
  }

  function loadFile(file) {
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => {
      markdown = String(reader.result ?? '');
    };
    reader.readAsText(file);
  }

  function onDrop(e) {
    e.preventDefault();
    dragOver = false;
    loadFile(e.dataTransfer?.files?.[0]);
  }
</script>

<div
  class="editor"
  class:dragover={dragOver}
  ondragover={(e) => {
    e.preventDefault();
    dragOver = true;
  }}
  ondragleave={() => (dragOver = false)}
  ondrop={onDrop}
  role="region"
  aria-label="Markdown input"
>
  <div class="toolbar">
    <button type="button" onclick={pickFile}>Open .md</button>
    <button type="button" onclick={() => (markdown = '')} disabled={!markdown}>
      Clear
    </button>
    <span class="hint">…or drop a file anywhere here</span>
    <span class="stats">{lineCount} lines · {charCount} chars</span>
  </div>
  <textarea
    bind:value={markdown}
    placeholder="# Hello&#10;&#10;Write Markdown here, then press Compile."
    spellcheck="false"
    aria-label="Markdown source"
  ></textarea>
  <input
    bind:this={fileInput}
    type="file"
    accept=".md,.markdown,.txt,text/markdown"
    hidden
    onchange={onFileChosen}
  />
</div>

<style>
  .editor {
    display: flex;
    flex-direction: column;
    min-height: 0;
    border: 1px solid #ddd;
    border-radius: 8px;
    overflow: hidden;
    background: #fff;
  }
  .editor.dragover {
    outline: 2px dashed #4a7ddb;
    outline-offset: -2px;
  }
  textarea {
    flex: 1;
    min-height: 320px;
    resize: none;
    border: 0;
    padding: 0.75rem;
    font-family: ui-monospace, monospace;
    font-size: 0.9rem;
    line-height: 1.5;
  }
  textarea:focus {
    outline: none;
  }
</style>
