interface PCDEditorOptions {
  wasmPath?: string
  wasmExecPath?: string
  idPrefix?: string
  logId?: string
  canvasId?: string
  onKeyDownHook?: (KeyboardEvent) => void
}

declare class PCDEditor {
  constructor(opts: PCDEditorOptions)
  attach(): Promise<null>
  appendDefaultMenuboxTo(selector: string): void
  loadPCD(path: string): Promise<null>
  loadSubPCD(path: string): Promise<null>
  load2D(yamlPath: string, imgPath: string): Promise<null>

  logger(any): void
  private qs: (q: string) => Element
  private wrapId: (q: string) => string

  pcdeditor: {
    reset(): void
    importPCD(a: Blob): Promise<string>
    importSubPCD(a: Blob): Promise<string>
    import2D(a, b: Blob): Promise<string>
    exportPCD(): Promise<Blob>
    exportSelectedPCD(): Promise<Blob>
    command(cmd: string): Promise<number[][]>
    show2D(show: boolean): Promise<string>
    exit(): void
  }

  opts: PCDEditorOptions

  canvas: HTMLCanvasElement

  log: HTMLDivElement

  localClipboard: Blob
}

export default PCDEditor
