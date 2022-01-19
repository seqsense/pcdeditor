import React, { useEffect } from 'react'

import PCDEditor from '../'

type AttachedCallback = (editor: PCDEditor) => void

type Props = {
  wasmPath: string
  wasmExecPath: string
  onAttached: AttachedCallback
  appendDefaultMenu?: boolean
  customMenu?: React.ReactNode
  idPrefix?: string
}

const Editor: React.FC<Props> = ({
  wasmPath,
  wasmExecPath,
  onAttached,
  appendDefaultMenu = true,
  customMenu,
  idPrefix = '',
}: Props) => {
  const logId = `${idPrefix}log`
  const canvasId = `${idPrefix}mapCanvas`
  const menuboxId = `${idPrefix}menubox`

  useEffect(() => {
    let cleanup = () => {
      // Cleanup function will be set after initialization.
    }

    const editor = new PCDEditor({
      wasmPath: wasmPath,
      wasmExecPath: wasmExecPath,
      idPrefix: idPrefix,
      canvasId: `#${canvasId}`,
      logId: `#${logId}`,
    })
    if (
      appendDefaultMenu &&
      document.querySelector(`#${idPrefix}exportPCD`) === null
    ) {
      editor.appendDefaultMenuboxTo(`#${menuboxId}`)
    }
    editor.attach().then(() => {
      onAttached(editor)
      cleanup = editor.pcdeditor.exit
    })

    return () => {
      cleanup()
    }
  }, [wasmPath, wasmExecPath, appendDefaultMenu])

  return (
    <div
      style={{
        borderWidth: 0,
        width: '100%',
        height: '100%',
        position: 'relative',
        color: 'black',
        overflow: 'visible',
      }}
    >
      <canvas
        id={canvasId}
        tabIndex={0}
        style={{
          width: '100%',
          height: '100%',
          touchAction: 'none',
        }}
      />
      <div
        id={menuboxId}
        className="pcdeditorMenubox"
        style={{
          display: 'flex',
          flexWrap: 'wrap',
          alignItems: 'stretch',
          position: 'absolute',
          top: 0,
          left: 0,
          padding: '2px',
          boxSizing: 'border-box',
        }}
      >
        {customMenu}
      </div>
      <div
        id={logId}
        className="pcdeditorLog"
        style={{
          fontSize: '0.6em',
          maxHeight: '20em',
          position: 'absolute',
          bottom: '1em',
          left: '1em',
          zIndex: 2,
          color: 'white',
          pointerEvents: 'none',
          overflow: 'hidden',
        }}
      />
    </div>
  )
}

export default Editor
