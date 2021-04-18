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
    }
    if (opts) {
      Object.keys(opts).forEach((key) => {
        this.opts[key] = opts[key]
      })
    }
    this.canvas = qsRaw(this.opts.canvasId)

    const log = qsRaw(this.opts.logId)
    this.logger = (msg) => {
      if (msg.toString !== undefined) {
        log.innerHTML = `${msg.toString().replace(/\n/g, '<br/>')}<br/>${
          log.innerHTML
        }`
      }
    }
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

  attach() {
    return new Promise((resolve) => {
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
          }
          if (e.keyCode === 27) {
            this.canvas.focus()
          }
        }
        this.qs('#show2D').onchange = (e) =>
          pcdeditor.show2D(e.target.checked).catch(this.logger)
        pcdeditor.show2D(this.qs('#show2D').checked).catch(this.logger)

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
      }
      /** main */
      const loadWasm = async () => {
        document.onPCDEditorLoaded = async (e) => {
          this.pcdeditor = e.attach(this.canvas, { logger: this.logger })
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
  ${selector} button, ${selector} input[type=text], ${selector} a {
    background-color: #ccc;
    border: none;
    min-height: 1.5em;
  }
  ${selector} button:hover, ${selector} input:hover, ${selector} a:hover, ${selector} span:not(.foldMenu):hover {
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
  }
  ${selector} span${id('.foldMenuIcon')}, ${selector} span${id(
      '.foldMenuHeader',
    )} {
    height: ${menuHeight};
    float: left;
    display: inline-flex;
    align-items: center;
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
  ${selector} ${id('.foldMenuElem')} button, ${selector} ${id(
      '.foldMenuElem',
    )} input {
    flex-grow: 2;
    flex-shrink: 2;
    margin: 2px;
    z-index: 2000;
    max-width: calc(100% - 4px);
  }
  ${selector} ${id('.inputLabel')} {
    width: 100%;
    margin-bottom: -2px;
    display: block;
    font-size: 0.875em;
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
        <g><path d="M24,12.03c0,0-5.37,7.98-12,7.98c-6.63,0-12-7.98-12-7.98s5.37-7.98,12-7.98C18.63,4.06,24,12.03,24,12.03z M17.83,12.03
          c0-3.22-2.61-5.83-5.83-5.83s-5.83,2.61-5.83,5.83s2.61,5.83,5.83,5.83S17.83,15.25,17.83,12.03z" /></g>
        <path d="M12,7.69c-0.48,0-0.94,0.1-1.37,0.24c0.4,0.33,0.65,0.82,0.65,1.38c0,1-0.81,1.81-1.81,1.81c-0.61,0-1.15-0.3-1.48-0.77
          c-0.22,0.52-0.34,1.08-0.34,1.67c0,2.4,1.95,4.34,4.34,4.34s4.34-1.95,4.34-4.34S14.4,7.69,12,7.69z" />
      </svg>
    </span><span class="${id('foldMenuHeader')}">View</span>
    <div class="${id('foldMenuElem')}"><button id="${id(
      'crop',
    )}">Crop/Uncrop</button></div>
    <hr />
    <div class="${id('foldMenuElem')}"><button id="${id(
      'side',
    )}">Side view</button></div>
    <div class="${id('foldMenuElem')}"><button id="${id(
      'top',
    )}">Top view</button></div>
    <div class="${id('foldMenuElem')}">
      <button id="${id('yaw90ccw')}" class="${id('twoButtons')}">←90°</button>
      <button id="${id('yaw90cw')}" class="${id('twoButtons')}">90°→</button>
    </div>
    <hr />
    <div class="${id('foldMenuElem')}">
      <label class="${id('inputLabel')}">View preset</label>
      <button id="${id('viewPresetReset')}" class="${id(
      'twoButtons',
    )}">Reset</button>
      <button id="${id('viewPresetFPS')}" class="${id(
      'twoButtons',
    )}">FPS</button>
    </div>
    <hr />
    <div class="${id('foldMenuElem')}">
      <label for="${id('pointSize')}" class="${id(
      'inputLabel',
    )}">Point size</label>
      <input type="range" id="${id(
        'pointSize',
      )}" min="20" max="100" value="40" />
    </div>
    <hr />
    <div class="${id('foldMenuElem')}">
      <label class="${id('inputLabel')}">Depth</label>
      <button id="${id('fovInc')}" class="${id(
      'twoButtons',
    )}"><svg width="1em" height="1em" viewBox="0 0 100 100">
        <path d="M 0 30 L 50 100 L 100 30 Q 50 -10 0 30 z" />
      </svg></button>
      <button id="${id('fovDec')}" class="${id(
      'twoButtons',
    )}"><svg width="1em" height="1em" viewBox="0 0 100 100">
        <path d="M 30 20 L 50 100 L 70 20 Q 50 0 30 20 z" />
      </svg></button>
    </div>
  </div>
</span>
<a href="https://github.com/seqsense/pcdeditor#%E6%93%8D%E4%BD%9C" target="_blank">?</a>
<button id="${id('resetContext')}">reset</button>
`
  }
}

module.exports = PCDEditor
