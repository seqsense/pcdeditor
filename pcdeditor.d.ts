interface PCDEditorOptions {
  wasmPath?: string
  wasmExecPath?: string
  idPrefix?: string
  logId?: string
  canvasId?: string
}

declare class PCDEditor {
  constructor(opts: PCDEditorOptions)
  attach(): Promise<null>
  appendDefaultMenuboxTo(selector: string): void

  logger(any): void
  private qs: (q: string) => Element
  private wrapId: (q: string) => string

  pcdeditor: {
    exportPCD(): Promise<Blob>
    loadPCD(a: string): Promise<null>
    load2D(a, b: string): Promise<null>
    exit(): void
  }

  opts: PCDEditorOptions

  canvas: HTMLCanvasElement

  log: HTMLDivElement
}

export default PCDEditor
