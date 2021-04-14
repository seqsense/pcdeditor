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
}

const Editor: React.FC<Props> = ({
  wasmPath,
  wasmExecPath,
  onAttached,
  appendDefaultMenu = true,
  customMenu,
}: Props) => {
  useEffect(() => {
    let cleanup = () => {
      // Cleanup function will be set after initialization.
    }

    const editor = new PCDEditor({
      wasmPath: wasmPath,
      wasmExecPath: wasmExecPath,
    })
    if (appendDefaultMenu && document.querySelector('#exportPCD') === null) {
      PCDEditor.appendDefaultMenuboxTo('#menubox')
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
      <canvas id="mapCanvas" tabIndex={0} />
      <div id="menubox">{customMenu}</div>
      <div id="log" />
    </Container>
  )
}

const Container = styled.div`
  border-width: 0;
  width: 100%;
  height: 100%;
  position: relative;
  color: black;

  canvas#mapCanvas {
    width: 100%;
    height: 100%;
    z-index: 1;
    touch-action: none;
  }
  div#menubox {
    z-index: 2;
    position: absolute;
    top: 0;
    left: 0;
    padding: 2px;
    box-sizing: border-box;
  }
  div#log {
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
