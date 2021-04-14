/** Shorthand for document.querySelector */
const qs = (q) => document.querySelector(q)

const fetchOpts = {
  credentials: 'include',
  cache: 'no-cache',
}

class PCDEditor {
  constructor(opts) {
    if (opts) {
      this.wasmPath = opts.wasmPath
      this.wasmExecPath = opts.wasmExecPath
    }
    if (!this.wasmPath) {
      this.wasmPath = 'pcdeditor.wasm'
    }
    if (!this.wasmExecPath) {
      this.wasmExecPath = 'wasm_exec.js'
    }
    this.logger = (msg) => {
      if (msg.toString !== undefined) {
        const log = qs('#log')
        log.innerHTML = `${msg.toString().replace(/\n/g, '<br/>')}<br/>${
          log.innerHTML
        }`
      }
    }
  }

  attach() {
    return new Promise((resolve) => {
      /** Sets up the control's event handlers */
      const setupControls = async (pcdeditor) => {
        qs('#exportPCD').onclick = async () => {
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
        qs('#command').onkeydown = (e) => {
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
            qs('#mapCanvas').focus()
          }
        }
        qs('#show2D').onchange = (e) =>
          pcdeditor.show2D(e.target.checked).catch(this.logger)
        pcdeditor.show2D(qs('#show2D').checked).catch(this.logger)

        const fovIncButton = qs('#fovInc')
        const fovDecButton = qs('#fovDec')
        const pointSizeInput = qs('#pointSize')

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
        qs('#ortho').onchange = (e) => projectionMode(e.target)
        projectionMode(qs('#ortho'))

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

        qs('#top').onclick = async () => {
          try {
            await pcdeditor.command('snap_yaw')
            await pcdeditor.command('pitch 0')
          } catch (e) {
            this.logger(e)
          }
        }
        qs('#side').onclick = async () => {
          try {
            await pcdeditor.command('snap_yaw')
            await pcdeditor.command('pitch 1.570796327')
          } catch (e) {
            this.logger(e)
          }
        }
        qs('#yaw90cw').onclick = () =>
          pcdeditor.command('rotate_yaw 1.570796327').catch(this.logger)
        qs('#yaw90ccw').onclick = () =>
          pcdeditor.command('rotate_yaw -1.570796327').catch(this.logger)
        qs('#crop').onclick = () => pcdeditor.command('crop').catch(this.logger)
        qs('#viewPresetReset').onclick = () =>
          pcdeditor.command('view_reset').catch(this.logger)
        qs('#viewPresetFPS').onclick = () =>
          pcdeditor.command('view_fps').catch(this.logger)

        qs('#resetContext').onclick = () => {
          const gl = qs('#mapCanvas').getContext('webgl2')
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
          this.pcdeditor = e.attach(qs('#mapCanvas'), { logger: this.logger })
          setupControls(this.pcdeditor)
          resolve()
        }
        const go = new global.Go()
        const { instance } = await WebAssembly.instantiateStreaming(
          fetch(this.wasmPath, { cache: 'no-cache' }),
          go.importObject,
        )
        go.run(instance)
      }
      if (typeof global === 'undefined' || typeof global.Go === 'undefined') {
        const script = document.createElement('script')
        script.onload = loadWasm
        script.src = this.wasmExecPath
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

  static appendDefaultMenuboxTo(selector) {
    qs(selector).innerHTML += `
<style>
  ${selector}>#command {
    opacity: 0.8;
  }
  ${selector}>span {
    background-color: #CCC;
    padding: 2px;
    border-radius: 0.3em;
    margin-left: 0.5em;
    position: relative;
  }
  ${selector} label {
    padding-left: 0.5em;
    padding-right: 0.2em;
    margin-left: -0.5em;
  }
  ${selector}>span.foldMenu:after {
    display: inline-block;
    width: 1em;
    height: 1em;
    content: "";
  }
  ${selector}>span.foldMenu>div:before {
    width: 100%;
    height: 100%;
    padding: 0 2em 2em;
    border-radius: 0 0 2em 2em;
    position: absolute;
    top: 0;
    left: -2em;
    content: "";
  }
  ${selector}>span.foldMenu>div {
    padding: 0;
    margin: 0;
    display: inline-block;
    width: 1em;
    height: 100%;
    overflow: hidden;
    position: absolute;
    top: 0;
    background-color: rgba(204, 204, 204, 0.9);
    border-radius: 0.3em;
    color: #666;
  }
  ${selector}>span.foldMenu>div:hover {
    width: 10em;
    height: auto;
    padding-bottom: 0.5em;
    overflow: visible;
  }
  ${selector}>span.foldMenu>div:after {
    content: "";
    display: block;
    clear: both;
  }
  ${selector}>span.foldMenu>div>span {
    cursor: pointer;
    color: #000;
    line-height: 100%;
  }
  ${selector}>span.foldMenu>div>span.foldMenuHeader {
    cursor: default;
    font-size: 0.75em;
    color: #333;
  }
  ${selector}>span.foldMenu>div>.foldMenuElem {
    color: #000;
    margin: 0 2px;
    width: calc(100% - 4px);
    position: relative;
  }
  ${selector}>span.foldMenu>div>.foldMenuElem:after {
    height: 2px;
    content: "";
  }
  ${selector} .twoButtons {
    margin: 0;
    width: 50%;
  }
  ${selector} .incDec>.twoButtons {
    width: auto;
  }
  ${selector} .incDec {
    width: 100%;
    text-align: center;
    line-height: 100%;
  }
  ${selector} .incDec>.twoButtons:first-child {
    float: left;
  }
  ${selector} .incDec>.twoButtons:last-child {
    float: right;
  }
  ${selector} .incDec>.twoButtons:disabled>svg {
    fill: #666;
  }
  ${selector} .incDec>.twoButtons:disabled + span.inputLabel {
    color: #666;
  }
  ${selector} .inputLabel{
    font-size: 0.75em;
  }
  ${selector} label.inputLabel{
    display: block;
    padding: 0 2px !important;
    margin: 0 !important;
  }
  ${selector}>span.foldMenu>div>hr {
    background-color: #AAA;
    height: 2px;
    border: none;
    margin: 8px 2px 4px 2px;
  }
  ${selector} .foldMenuFill{
    width: calc(100% - 4px);
  }
  ${selector}>a {
    text-decoration: none;
    color: black;
    padding: 2px 0.5em;
    background-color: rgba(255, 255, 255, 0.8);
    border-radius: 0.3em;
    margin-left: 0.5em;
  }
</style>
<button id="exportPCD">export</button>
<input type="text" id="command" />
<span>
  <input type="checkbox" checked id="show2D" />
  <label for="show2D">2D</label>
</span>
<span>
  <input type="checkbox" checked id="ortho" />
  <label for="ortho">Ortho</label>
</span>
<span class="foldMenu">
  <div>
    <span>👁</span> <span class="foldMenuHeader">View</span>
    <button id="crop" class="foldMenuElem">Crop/Uncrop</button>
    <hr />
    <button id="side" class="foldMenuElem">Side view</button>
    <button id="top" class="foldMenuElem">Top view</button>
    <div class="foldMenuElem">
      <button id="yaw90ccw" class="twoButtons">←90°</button><button id="yaw90cw" class="twoButtons">90°→</button>
    </div>
    <hr />
    <div class="foldMenuElem">
      <label for="pointSize" class="inputLabel">Point size</label>
      <input type="range" id="pointSize" class="foldMenuFill" min="20" max="100" value="40" />
    </div>
    <hr />
    <div class="foldMenuElem">
      <label class="inputLabel">View preset</label>
      <button id="viewPresetReset" class="twoButtons">Reset</button><button id="viewPresetFPS" class="twoButtons">FPS</button>
    </div>
    <hr />
    <div class="foldMenuElem incDec">
      <button id="fovInc" class="twoButtons"><svg width="1em" height="1em" viewBox="0 0 100 100">
        <path d="M 0 30 L 50 100 L 100 30 Q 50 -10 0 30 z" />
      </svg></button>
      <span class="inputLabel">Depth</span>
      <button id="fovDec" class="twoButtons"><svg width="1em" height="1em" viewBox="0 0 100 100">
        <path d="M 30 20 L 50 100 L 70 20 Q 50 0 30 20 z" />
      </svg></button>
    </div>
  </div>
</span>
<a href="https://github.com/seqsense/pcdeditor#%E6%93%8D%E4%BD%9C" target="_blank">❓</a>
<button id="resetContext">reset</button>
`
  }
}

module.exports = PCDEditor
