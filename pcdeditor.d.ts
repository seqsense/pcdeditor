declare class PCDEditor {
  constructor(opts: { wasmPath: string; wasmExecPath: string })
  attach(): Promise<null>
  logger(any): void
  static appendDefaultMenuboxTo(selector: string): void

  pcdeditor: {
    exportPCD(): Promise<Blob>
    loadPCD(a: string): Promise<null>
    load2D(a, b: string): Promise<null>
    exit(): void
  }

  wasmPath: string

  wasmExecPath: string
}

export = PCDEditor
