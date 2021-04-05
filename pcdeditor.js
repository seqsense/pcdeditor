/* global global, Go, Postmate */

{
  /** Shorthand for document.querySelector */
  const qs = (q) => document.querySelector(q);

  global.PCDEditor = class {
    constructor () {
      this.logger = msg => {
        if (msg.toString != undefined) {
          const log = qs('#log');
          log.innerHTML = msg.toString().replace(/\n/g, '<br/>') + '<br/>' + log.innerHTML;
        }
      };
    }

    attach () {
      const that = this;
      return new Promise(resolve => {
        /** Sets up the control's event handlers */
        const setupControls = async pcdeditor => {
          qs('#exportPCD').onclick = async () => {
            try {
              const blob = await pcdeditor.exportPCD()
              const a = document.createElement('a');
              a.download = 'exported.pcd';
              a.href = URL.createObjectURL(blob);
              a.dataset.downloadurl = ['application/octet-stream', a.download, a.href];
              a.click();
            } catch (e) {
              that.logger(e);
            }
          };
          qs('#command').onkeydown = (e) => {
            if (e.keyCode == 13) {
              pcdeditor.command(e.target.value).then(res => {
                let str = '';
                for (const vec of res) {
                  for (const val of vec) {
                    str += val.toFixed(3) + ' ';
                  }
                  str += '\n';
                }
                if (str != '') {
                  that.logger(str.trim())
                }
              }).catch(that.logger);
              e.target.value = '';
            }
            if (e.keyCode == 27) {
              qs('#mapCanvas').focus();
            }
          };
          qs('#show2D').onchange =
            (e) => pcdeditor.show2D(e.target.checked).catch(that.logger);
          pcdeditor.show2D(qs('#show2D').checked).catch(that.logger);

          const projectionMode =
            (target) =>
            pcdeditor.command(target.checked ? 'ortho' : 'perspective').catch(that.logger);
          qs('#ortho').onchange = e => projectionMode(e.target);
          projectionMode(qs('#ortho'));

          qs('#top').onclick = async () => {
            try {
              await pcdeditor.command('snap_yaw');
              await pcdeditor.command('pitch 0');
            } catch (e) {
              that.logger(e);
            }
          };
          qs('#side').onclick = async () => {
            try {
              await pcdeditor.command('snap_yaw');
              await pcdeditor.command('pitch 1.570796327');
            } catch (e) {
              that.logger(e);
            }
          };
          qs('#yaw90cw').onclick = () => pcdeditor.command('rotate_yaw 1.570796327').catch(that.logger);
          qs('#yaw90ccw').onclick = () => pcdeditor.command('rotate_yaw -1.570796327').catch(that.logger);
          qs('#crop').onclick = () => pcdeditor.command('crop').catch(that.logger);
          qs('#pointSize').onchange = () => {
            const val = qs('#pointSize').value;
            pcdeditor.command(`point_size ${val}`).catch(that.logger);
          };

          qs('#resetContext').onclick = () => {
            const gl = qs('#mapCanvas').getContext('webgl2');
            const glex = gl.getExtension('WEBGL_lose_context');
            glex.loseContext();
            const retryRestore = setInterval(() => {
              try {
                glex.restoreContext();
              } catch (error) {
                return;
              }
              clearInterval(retryRestore);
            }, 1000);
          };
        };
        /** main */
        window.onload = async () => {
          document.onPCDEditorLoaded = async (e) => {
            that.pcdeditor = e.attach(qs('#mapCanvas'), { logger: that.logger });
            setupControls(that.pcdeditor);
            resolve();
          };
          const go = new Go();
          const { instance } = await WebAssembly.instantiateStreaming(fetch('pcdeditor.wasm', {cache: "no-cache"}), go.importObject)
          go.run(instance);
        }
      });
    }

    async initSavePCD () {
      const that = this;
      const setupControls = async (pcdeditor, parentFrame) => {
        const saveBtn = qs('#savePCD')
        saveBtn.onclick = async () => {
          try {
            const blob = await pcdeditor.exportPCD();
            const objectURL = URL.createObjectURL(blob);
            parentFrame.emit('save-pcd', objectURL);
            saveBtn.disabled = true;
          } catch (e) {
            that.logger(e);
          }
        };
      };
      /** Sets up the postmate (iframe communication library) */
      const setupPostmate = (pcdeditor) => new Postmate.Model({
        load: async ({ mapPcd, mapYaml, mapPng }) => {
          if (!mapPcd) {
            that.logger('map.pcd is not given');
            return;
          }
          pcdeditor.loadPCD(mapPcd).catch(that.logger);
          if (!mapYaml) {
            that.logger('map.yaml is not given');
            return;
          }
          if (!mapPng) {
            that.logger('map.png is not given');
            return;
          }
          pcdeditor.load2D(mapYaml, mapPng).catch(that.logger);
        },
        enableSaveBtn: () => {
          qs('#savePCD').disabled = false
        }
      });
      const parentFrame = await setupPostmate(that.pcdeditor);
      setupControls(that.pcdeditor, parentFrame);
    }
  };
}
