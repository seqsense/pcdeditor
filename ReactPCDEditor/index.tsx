import React, { useEffect } from 'react'
import styled from 'styled-components'

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
    <Container>
      <canvas id={canvasId} tabIndex={0} />
      <div id={menuboxId} className="pcdeditorMenubox">
        {customMenu}
      </div>
      <div id={logId} className="pcdeditorLog" />
    </Container>
  )
}

const Container = styled.div`
  border-width: 0;
  width: 100%;
  height: 100%;
  position: relative;
  color: black;
  overflow: visible;

  canvas {
    width: 100%;
    height: 100%;
    touch-action: none;
  }
  div.pcdeditorMenubox {
    display: flex;
    flex-wrap: wrap;
    align-items: stretch;
    position: absolute;
    top: 0;
    left: 0;
    padding: 2px;
    box-sizing: border-box;
  }
  div.pcdeditorLog {
    font-size: 0.6em;
    max-height: 20em;
    position: absolute;
    bottom: 1em;
    left: 1em;
    z-index: 2;
    color: white;
    pointer-events: none;
    overflow: hidden;
  }
`

export default Editor
