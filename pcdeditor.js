/** Shorthand for document.querySelector */
const qsRaw = (q) => document.querySelector(q)

const fetchOpts = {
  credentials: 'include',
  cache: 'no-cache',
}

class PCDEditor {
  constructor(opts) {
    this.opts = {
      wasmPath: 'pcdeditor.wasm',
      wasmExecPath: 'wasm_exec.js',
      idPrefix: '',
      logId: '#log',
      canvasId: '#mapCanvas',
      onKeyDownHook: (e) => {
        if (e.ctrlKey) {
          switch (e.code) {
          case 'KeyC':
            this.qs('#clipboardCopy').click()
            break
          case 'KeyV':
            this.qs('#clipboardPaste').click()
            break
          }
        }
      },
    }
    if (opts) {
      Object.keys(opts).forEach((key) => {
        this.opts[key] = opts[key]
      })
    }
    this.canvas = qsRaw(this.opts.canvasId)

    const log = qsRaw(this.opts.logId)
    this.logger = (msg) => {
      if (msg?.toString) {
        log.innerHTML = `${msg.toString().replace(/\n/g, '<br/>')}<br/>${
          log.innerHTML
        }`
      }
    }

    this.localClipboard = new Blob()
  }

  wrapId(q) {
    if (this.opts.idPrefix !== '') {
      switch (q[0]) {
        case '#':
          q = q.replace('#', `#${this.opts.idPrefix}`)
          break
        case '.':
          q = q.replace('.', `.${this.opts.idPrefix}`)
          break
        default:
          q = `${this.opts.idPrefix}${q}`
          break
      }
    }
    return q
  }

  qs(q) {
    return document.querySelector(this.wrapId(q))
  }

  qsAll(q) {
    return document.querySelectorAll(this.wrapId(q))
  }

  attach() {
    return new Promise((resolve) => {
      const backdrop = document.createElement('div')
      backdrop.id = `${this.wrapId('foldMenuBackdrop')}`
      this.canvas.parentNode.insertBefore(backdrop, this.canvas.nextSibling)
      const busyBackdrop = document.createElement('div')
      busyBackdrop.id = `${this.wrapId('busyBackdrop')}`
      this.canvas.parentNode.insertBefore(busyBackdrop, this.canvas.nextSibling)

      /** Sets up the control's event handlers */
      const setupControls = async (pcdeditor) => {
        this.qs('#exportPCD').onclick = async () => {
          try {
            const blob = await pcdeditor.exportPCD()
            const a = document.createElement('a')
            a.download = 'exported.pcd'
            a.href = URL.createObjectURL(blob)
            a.dataset.downloadurl = [
              'application/octet-stream',
              a.download,
              a.href,
            ]
            a.click()
          } catch (e) {
            this.logger(e)
          }
        }
        this.qs('#command').onkeydown = (e) => {
          if (e.keyCode === 13) {
            pcdeditor
              .command(e.target.value)
              .then((res) => {
                let str = ''
                // eslint-disable-next-line no-restricted-syntax
                for (const vec of res) {
                  // eslint-disable-next-line no-restricted-syntax
                  for (const val of vec) {
                    str += `${val.toFixed(3)} `
                  }
                  str += '\n'
                }
                if (str !== '') {
                  this.logger(str.trim())
                }
              })
              .catch(this.logger)
            e.target.value = ''
            this.canvas.focus()
          }
          if (e.keyCode === 27) {
            this.canvas.focus()
          }
        }
        this.qs('#show2D').onchange = (e) =>
          pcdeditor.show2D(e.target.checked).catch(this.logger)
        pcdeditor.show2D(this.qs('#show2D').checked).catch(this.logger)

        // View menu
        const fovIncButton = this.qs('#fovInc')
        const fovDecButton = this.qs('#fovDec')
        const pointSizeInput = this.qs('#pointSize')

        const projectionMode = (target) => {
          if (target.checked) {
            fovDecButton.disabled = true
            fovIncButton.disabled = true
            pcdeditor.command('ortho').catch(this.logger)
          } else {
            fovDecButton.disabled = false
            fovIncButton.disabled = false
            pcdeditor.command('perspective').catch(this.logger)
          }
        }
        this.qs('#ortho').onchange = (e) => projectionMode(e.target)
        projectionMode(this.qs('#ortho'))

        const onPointSizeChange = (target) => {
          const val = target.value
          pcdeditor.command(`point_size ${val}`).catch(this.logger)
        }
        pointSizeInput.oninput = (e) => onPointSizeChange(e.target)
        pointSizeInput.onchange = (e) => onPointSizeChange(e.target)
        onPointSizeChange(pointSizeInput)

        fovDecButton.onclick = () =>
          pcdeditor.command('fov -1').catch(this.logger)
        fovIncButton.onclick = () =>
          pcdeditor.command('fov 1').catch(this.logger)

        this.qs('#top').onclick = async () => {
          try {
            await pcdeditor.command('snap_yaw')
            await pcdeditor.command('pitch 0')
          } catch (e) {
            this.logger(e)
          }
        }
        this.qs('#side').onclick = async () => {
          try {
            await pcdeditor.command('snap_yaw')
            await pcdeditor.command('pitch 1.570796327')
          } catch (e) {
            this.logger(e)
          }
        }
        this.qs('#yaw90cw').onclick = () =>
          pcdeditor.command('rotate_yaw 1.570796327').catch(this.logger)
        this.qs('#yaw90ccw').onclick = () =>
          pcdeditor.command('rotate_yaw -1.570796327').catch(this.logger)
        this.qs('#crop').onclick = () =>
          pcdeditor.command('crop').catch(this.logger)
        this.qs('#viewPresetReset').onclick = () =>
          pcdeditor.command('view_reset').catch(this.logger)
        this.qs('#viewPresetFPS').onclick = () =>
          pcdeditor.command('view_fps').catch(this.logger)

        // Select menu
        const selThickLogInput = this.qs('#selThickLog')

        this.qs('#unselect').onclick = () =>
          pcdeditor.command('unset_cursor').catch(this.logger)
        this.qs('#vsnap').onclick = () =>
          pcdeditor.command('snap_v').catch(this.logger)
        this.qs('#hsnap').onclick = () =>
          pcdeditor.command('snap_h').catch(this.logger)

        const onSelThickLogChange = (target) => {
          const val = Math.pow(target.value, 2)
          pcdeditor.command(`select_range_perspective ${val}`).catch(this.logger)
        }
        selThickLogInput.oninput = (e) => onSelThickLogChange(e.target)
        selThickLogInput.onchange = (e) => onSelThickLogChange(e.target)
        onSelThickLogChange(selThickLogInput)

        // Edit menu
        const surfaceGridInput = this.qs('#surfaceGrid')
        const labelIdInput = this.qs('#labelID')

        this.qs('#undo').onclick = () =>
          pcdeditor.command('undo').catch(this.logger)
        this.qs('#createSurface').onclick = () => {
          const grid = surfaceGridInput.value
          pcdeditor.command(`add_surface ${grid}`).catch(this.logger)
        }
        this.qs('#unsetLabel').onclick = () =>
          pcdeditor.command('label 0').catch(this.logger)
        this.qs('#setLabel').onclick = () => {
          const id = labelIdInput.value
          pcdeditor.command(`label ${id}`).catch(this.logger)
        }
        this.qs('#delete').onclick = () =>
          pcdeditor.command('delete').catch(this.logger)
        const insertSubPcdFile = this.qs('#insertSubPcdFile')
        this.qs('#insertSubPcd').onclick = async () =>
          insertSubPcdFile.click()
        insertSubPcdFile.onchange = async (e) =>
          this.loadSubPCD(URL.createObjectURL(e.target.files[0]))
            .finally(() => {
              insertSubPcdFile.value = ''
              this.canvas.focus()
            })
        this.qs('#clipboardCopy').onclick = async () => {
          this.canvas.focus()
          try {
            const blob = await pcdeditor.exportSelectedPCD()
            // Clipboard can't be used on insecure context
            this.localClipboard = blob

            if (!navigator?.clipboard?.writeText) {
              this.logger('clipped data is available only in this window')
              return
            }
            const fr = new FileReader()
            fr.onload = () => {
              navigator.clipboard.writeText(fr.result)
                .catch(() => this.logger('clipped data is available only in this window'))
            }
            fr.onabort = () => {
              this.logger('failed to encode data')
            }
            fr.readAsDataURL(blob)
          } catch (e) {
            this.logger(e)
          }
        }
        this.qs('#clipboardPaste').onclick = async () => {
          this.canvas.focus()
          try {
            if (!navigator?.clipboard?.readText) {
              // Fallback to the local data
              await this.loadSubPCD(URL.createObjectURL(this.localClipboard))
              return
            }
            const text = await navigator.clipboard.readText()
            if (text.startsWith('data:application/x-pcd;base64,')) {
              await this.loadSubPCD(text)
            }
          } catch (e) {
            this.logger(e)
          }
        }
        const fitInserting = (args) => {
          busyBackdrop.innerText = 'Processing'
          busyBackdrop.style.display = 'flex'
          setTimeout(() => {
            pcdeditor.command(`fit_inserting ${args}`).catch(this.logger)
            busyBackdrop.style.display = 'none'
            this.canvas.focus()
          }, 50)
        }
        this.qs('#fitInsertingXYZYaw').onclick = async () =>
          fitInserting('0 1 2 5')
        this.qs('#fitInsertingXYZ').onclick = async () =>
          fitInserting('0 1 2')

        // Debug menu
        this.qs('#resetContext').onclick = () => {
          const gl = this.canvas.getContext('webgl2')
          const glex = gl.getExtension('WEBGL_lose_context')
          glex.loseContext()
          const retryRestore = setInterval(() => {
            try {
              glex.restoreContext()
            } catch (error) {
              return
            }
            clearInterval(retryRestore)
          }, 1000)
        }

        let backdropDisableTimer
        this.qsAll('.foldMenu').forEach((e) => {
          e.onmouseenter = () => {
            if (backdropDisableTimer) {
              clearTimeout(backdropDisableTimer)
            }
            backdrop.style.display = 'block'
          }
          e.onmouseleave = () => {
            backdropDisableTimer = setTimeout(
              () => (backdrop.style.display = 'none'),
              0,
            )
          }
        })
      }
      /** main */
      const loadWasm = async () => {
        document.onPCDEditorLoaded = async (e) => {
          this.pcdeditor = e.attach(this.canvas, {
            logger: this.logger,
            onKeyDownHook: this.opts.onKeyDownHook,
          })
          setupControls(this.pcdeditor)
          resolve()
        }
        const go = new global.Go()
        const { instance } = await WebAssembly.instantiateStreaming(
          fetch(this.opts.wasmPath, { cache: 'no-cache' }),
          go.importObject,
        )
        go.run(instance)
      }
      if (typeof global === 'undefined' || typeof global.Go === 'undefined') {
        const script = document.createElement('script')
        script.onload = loadWasm
        script.src = this.opts.wasmExecPath
        document.head.appendChild(script)
      } else {
        loadWasm()
      }
    })
  }

  loadPCD(path) {
    return new Promise((resolve, reject) => {
      fetch(path, fetchOpts)
        .then((resp) => {
          if (!resp.ok) {
            reject(new Error(`failed to load map.pcd: ${resp.statusText}`))
            return undefined
          }
          return resp.blob()
        })
        .then((blob) => {
          return this.pcdeditor.importPCD(blob)
        })
        .then(() => {
          resolve()
        })
        .catch((e) => {
          reject(e)
        })
    })
  }

  loadSubPCD(path) {
    return new Promise((resolve, reject) => {
      fetch(path, fetchOpts)
        .then((resp) => {
          if (!resp.ok) {
            reject(new Error(`failed to load sub pcd: ${resp.statusText}`))
            return undefined
          }
          return resp.blob()
        })
        .then((blob) => {
          return this.pcdeditor.importSubPCD(blob)
        })
        .then(() => {
          resolve()
        })
        .catch((e) => {
          reject(e)
        })
    })
  }

  load2D(yamlPath, imgPath) {
    return new Promise((resolve, reject) => {
      fetch(yamlPath, fetchOpts)
        .then((resp) => {
          if (!resp.ok) {
            reject(new Error(`failed to load map.yaml: ${resp.statusText}`))
            return undefined
          }
          return resp.blob()
        })
        .then((yamlBlob) => {
          const img = new global.Image()
          img.onload = async () => {
            await this.pcdeditor.import2D(yamlBlob, img)
            resolve()
          }
          img.error = (e) => {
            reject(new Error(`failed to load map.yaml: ${e.toString()}`))
          }
          img.crossOrigin = 'use-credentials'
          img.src = imgPath
        })
        .catch((e) => {
          reject(e)
        })
    })
  }

  appendDefaultMenuboxTo(selector) {
    const menuHeight = '28px'
    const id = this.wrapId.bind(this)
    document.querySelector(selector).innerHTML += `
<style>
  ${selector} button, ${selector} input, ${selector} a, ${selector} span, ${selector} label {
    display: inline-block;
    font-size: 0.875rem;
    border-radius: 4px;
    box-sizing: border-box;
    margin: 0;
  }
  ${selector} input:focus {
    outline: none;
  }
  ${selector}>button, ${selector}>input, ${selector}>a, ${selector}>span, ${selector}>label {
    margin: 0 8px 4px 0;
    padding: 0 14px;
    height: ${menuHeight};
    display: inline-flex;
    align-items: center;
    justify-content: center;
  }
  ${selector}>button, ${selector}>input[type=text] {
    flex-grow: 2;
    max-width: 10em;
  }
  ${selector} button,
  ${selector} input[type=text],
  ${selector} input[type=number],
  ${selector} a {
    background-color: #ccc;
    border: none;
    min-height: 1.5em;
  }
  ${selector} button:hover, ${selector}>input:hover, ${selector}>a:hover, ${selector} span:not(.foldMenu):not(.foldMenuIcon):hover {
    background-color: #ddd;
    box-shadow: -1px -1px 1px #999 inset;
  }
  ${selector} button:active, ${selector} input:active, ${selector} a:active {
    box-shadow: 1px 1px 2px #999 inset;
  }
  ${selector} button:disabled, ${selector} input:disabled, ${selector} a:disabled {
    box-shadow: none;
    background-color: #aaa;
    color: #666;
  }
  ${selector} input:placeholder {
    color: #666;
  }
  ${selector} button:disabled svg {
    fill: #666;
  }
  ${selector}>${id('#command')} {
    opacity: 0.8;
  }
  ${selector}>span:not(${id('.foldMenu')}) {
    background-color: #ccc;
  }
  ${selector} label {
    padding-left: 4px;
  }
  ${selector}>span${id('.foldMenu')} {
    width: calc(1em + 16px);
    position: relative;
  }
  ${selector}>span${id('.foldMenu')}>div:before {
    width: 100%;
    height: 100%;
    padding: 0 2em 2em;
    border-radius: 0 0 2em 2em;
    position: absolute;
    top: 0;
    left: -2em;
    content: "";
    z-index: 2;
  }
  ${selector}>span${id('.foldMenu')}>div {
    box-sizing: content-box;
    position: absolute;
    padding: 0 8px;
    top: 0;
    left: 0;
    width: 1em;
    height: 100%;
    overflow: hidden;
    background-color: rgba(160, 160, 160, 0.9);
    border-radius: 4px;
  }
  ${selector}>span${id('.foldMenu')}>div:hover {
    width: 10em;
    height: auto;
    padding-bottom: 0.5em;
    overflow: visible;
    backdrop-filter: blur(1px);
    color: #222;
    z-index: 4;
  }
  ${selector} span${id('.foldMenuIcon')},
  ${selector} span${id('.foldMenuHeader')} {
    height: ${menuHeight};
    float: left;
    display: inline-flex;
    align-items: center;
    position: relative;
  }
  ${selector} span${id('.foldMenuIcon')}:after {
    position: absolute;
    width: 1em;
    height: 1em;
    z-index: 10;
    content: "";
  }
  ${selector} span${id('.foldMenuHeader')} {
    font-size: 0.875em;
    margin-left: 8px;
  }
  ${selector} ${id('.foldMenuElem')} {
    width: 100%;
    display: flex;
    flex-wrap: wrap;
  }
  ${selector} ${id('.foldMenuElem')} button,
  ${selector} ${id('.foldMenuElem')} input {
    flex-grow: 2;
    flex-shrink: 2;
    margin: 2px;
    z-index: 2000;
    max-width: calc(100% - 4px);
  }
  ${selector} ${id('.foldMenuElem')} input {
    border: none;
  }
  ${selector} ${id('.inputLabel')} {
    width: 100%;
    display: block;
    font-size: 0.875em;
  }
  ${selector} ${id('.inputLabelShort')} {
    display: flex;
    font-size: 0.875em;
    align-items: center;
    padding-right: 2px;
  }
  ${selector}>span${id('.foldMenu')}>div>hr {
    background-color: #aaa;
    height: 2px;
    border: none;
    margin: 4px 2px 4px 2px;
  }
  ${selector}>a {
    text-decoration: none;
    text-align: center;
    font-weight: 800;
  }
  ${id('#foldMenuBackdrop')} {
    display: none;
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background-color: rgba(0, 0, 0, 0.1);
  }
  ${id('#busyBackdrop')} {
    display: none;
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background-color: rgba(0, 0, 0, 0.3);
    color: white;
    align-items: center;
    justify-content: center;
    z-index: 2001;
    cursor: wait;
  }
</style>
<button id="${id('exportPCD')}">export</button>
<input type="text" id="${id('command')}" placeholder="command" />
<span>
  <input type="checkbox" checked id="${id('show2D')}" />
  <label for="${id('show2D')}">2D</label>
</span>
<span>
  <input type="checkbox" checked id="${id('ortho')}" />
  <label for="${id('ortho')}">Ortho</label>
</span>
<span class="${id('foldMenu')}">
  <div>
    <span class="${id('foldMenuIcon')}">
      <svg viewBox="0 0 24 24" width="1em" height="1em">
        <path d="M24,12.03c0,0-5.37,7.98-12,7.98c-6.63,0-12-7.98-12-7.98s5.37-7.98,12-7.98C18.63,4.06,24,12.03,24,12.03z M17.83,12.03
          c0-3.22-2.61-5.83-5.83-5.83s-5.83,2.61-5.83,5.83s2.61,5.83,5.83,5.83S17.83,15.25,17.83,12.03z" />
        <path d="M12,7.69c-0.48,0-0.94,0.1-1.37,0.24c0.4,0.33,0.65,0.82,0.65,1.38c0,1-0.81,1.81-1.81,1.81c-0.61,0-1.15-0.3-1.48-0.77
          c-0.22,0.52-0.34,1.08-0.34,1.67c0,2.4,1.95,4.34,4.34,4.34s4.34-1.95,4.34-4.34S14.4,7.69,12,7.69z" />
      </svg>
    </span><span class="${id('foldMenuHeader')}">View</span>
    <div class="${id('foldMenuElem')}">
      <button id="${id('crop')}">Crop/Uncrop</button>
    </div>
    <hr />
    <div class="${id('foldMenuElem')}">
      <button id="${id('side')}">Side view</button>
    </div>
    <div class="${id('foldMenuElem')}">
      <button id="${id('top')}">Top view</button>
    </div>
    <div class="${id('foldMenuElem')}">
      <button id="${id('yaw90ccw')}">←90°</button>
      <button id="${id('yaw90cw')}">90°→</button>
    </div>
    <hr />
    <div class="${id('foldMenuElem')}">
      <label class="${id('inputLabel')}">View preset</label>
      <button id="${id('viewPresetReset')}">Reset</button>
      <button id="${id('viewPresetFPS')}">FPS</button>
    </div>
    <hr />
    <div class="${id('foldMenuElem')}">
      <label
        for="${id('pointSize')}"
        class="${id('inputLabel')}"
      >Point size</label>
      <input
        id="${id('pointSize')}"
        type="range" min="20" max="100" value="40"
      />
    </div>
    <hr />
    <div class="${id('foldMenuElem')}">
      <label class="${id('inputLabel')}">Depth</label>
      <button id="${id('fovInc')}">
        <svg width="1em" height="1em" viewBox="0 0 100 100">
          <path d="M 0 30 L 50 100 L 100 30 Q 50 -10 0 30 z" />
        </svg>
      </button>
      <button id="${id('fovDec')}">
        <svg width="1em" height="1em" viewBox="0 0 100 100">
          <path d="M 30 20 L 50 100 L 70 20 Q 50 0 30 20 z" />
        </svg>
      </button>
    </div>
  </div>
</span>
<span class="${id('foldMenu')}">
  <div>
    <span class="${id('foldMenuIcon')}">
      <svg viewBox="0 0 24 24" width="1em" height="1em">
        <g stroke="#000" stroke-width="2" stroke-linejoin="round">
          <line x1="14.333" y1="1" x2="9.667" y2="1" />
          <polyline points="1,5 1,1 5,1" />
          <line x1="9.667" y1="23" x2="14.334" y2="23" />
          <polyline points="23,19 23,23 19,23" />
          <line x1="1" y1="9.667" x2="1" y2="14.334" />
          <polyline points="5,23 1,23 1,19" />
          <line x1="23" y1="14.333" x2="23" y2="9.666" />
          <polyline points="19,1 23,1 23,5" />
        </g>
      </svg>
    </span><span class="${id('foldMenuHeader')}">Select</span>
    <div class="${id('foldMenuElem')}">
      <button id="${id('unselect')}">Unselect</button>
    </div>
    <hr />
    <div class="${id('foldMenuElem')}">
      <button id="${id('vsnap')}">V Snap</button>
      <button id="${id('hsnap')}">H Snap</button>
    </div>
    <hr />
    <div class="${id('foldMenuElem')}">
      <label
        for="${id('selThickLog')}"
        class="${id('inputLabel')}"
      >Select surface thickness</label>
      <input
        id="${id('selThickLog')}"
        type="range" min="0.05" max="4" step="0.05" value="0.25"
      />
    </div>
  </div>
</span>
<span class="${id('foldMenu')}">
  <div>
    <span class="${id('foldMenuIcon')}">
      <svg viewBox="0 0 24 24" width="1em" height="1em">
        <polygon points="11.607,19.121 7.373,20.025 7.269,15.699 19.66,0 24,3.425 "/>
        <path fill="none" stroke="#000" stroke-width="2" d="M15.635,20.5c0,1.375-1.125,2.5-2.5,2.5H3.5C2.125,23,1,21.875,1,20.5v-17C1,2.125,2.125,1,3.5,1h9.635c1.375,0,2.5,1.125,2.5,2.5V20.5z"/>
      </svg>
    </span><span class="${id('foldMenuHeader')}">Edit</span>
    <div class="${id('foldMenuElem')}">
      <button id="${id('undo')}">Undo</button>
    </div>
    <hr/>
    <div class="${id('foldMenuElem')}">
      <div class="${id('foldMenuElem')}">
        <label
          for="${id('surfaceGrid')}"
          class="${id('inputLabelShort')}"
        >Grid</label>
        <input
          id="${id('surfaceGrid')}"
          type="number" value="0.05" min="0.01" step="0.01" max="1"
          style="width: 4em; text-align: center;"
        />
      </div>
      <button id="${id('createSurface')}">Create surface</button>
    </div>
    <hr/>
    <div class="${id('foldMenuElem')}">
      <label class="${id('inputLabel')}">Label</label>
      <div class="${id('foldMenuElem')}">
        <button id="${id('unsetLabel')}">Unset</button>
      </div>
      <div class="${id('foldMenuElem')}">
        <label
          for="${id('labelID')}"
          class="${id('inputLabelShort')}"
        >ID</label>
        <input
          id="${id('labelID')}"
          type="number" value="1" min="1" max="255"
          style="width: 3em; text-align: center;"
        />
        <button id="${id('setLabel')}">Set</button>
      </div>
    </div>
    <hr/>
    <div class="${id('foldMenuElem')}">
      <button id="${id('delete')}">Delete</button>
    </div>
    <hr/>
    <div class="${id('foldMenuElem')}">
      <label
        class="${id('inputLabelShort')}"
      >Insert pcd</label>
      <input
        id="${id('insertSubPcdFile')}"
        type="file"
        accept=".pcd"
        style="display: none;"
      />
      <button id="${id('insertSubPcd')}">Select file</button>
    </div>
    <div class="${id('foldMenuElem')}">
      <label class="${id('inputLabel')}">Clipboard</label>
      <button id="${id('clipboardCopy')}">Copy</button>
      <button id="${id('clipboardPaste')}">Paste</button>
    </div>
    <hr/>
    <div class="${id('foldMenuElem')}">
      <label class="${id('inputLabel')}">Fitting (experimental)</label>
      <button id="${id('fitInsertingXYZYaw')}">Fit X, Y, Z, Yaw</button>
      <button id="${id('fitInsertingXYZ')}">Fit X, Y, Z</button>
    </div>
  </div>
</span>
<a href="https://github.com/seqsense/pcdeditor#%E6%93%8D%E4%BD%9C" target="_blank">?</a>
<button id="${id('resetContext')}">reset</button>
`
  }
}

module.exports = PCDEditor
