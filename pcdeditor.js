/* global global, Go, Postmate */

{
  /** Shorthand for document.querySelector */
  const qs = (q) => document.querySelector(q);

  const fetchOpts = {
    credentials: 'include',
    cache: 'no-cache'
  };

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

          const fovIncButton = qs('#fovInc');
          const fovDecButton = qs('#fovDec');
          const pointSizeInput = qs('#pointSize');

          const projectionMode = (target) => {
            if (target.checked) {
              fovDecButton.disabled = true;
              fovIncButton.disabled = true;
              pcdeditor.command('ortho').catch(that.logger);
            } else {
              fovDecButton.disabled = false;
              fovIncButton.disabled = false;
              pcdeditor.command('perspective').catch(that.logger);
            }
          };
          qs('#ortho').onchange = e => projectionMode(e.target);
          projectionMode(qs('#ortho'));

          const onPointSizeChange = (target) => {
            const val = target.value;
            pcdeditor.command(`point_size ${val}`).catch(that.logger);
          };
          pointSizeInput.onchange = e => onPointSizeChange(e.target);
          onPointSizeChange(pointSizeInput);

          fovDecButton.onclick = () => pcdeditor.command('fov -1').catch(that.logger);
          fovIncButton.onclick = () => pcdeditor.command('fov 1').catch(that.logger);

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
          qs('#viewPresetReset').onclick = () => pcdeditor.command('view_reset').catch(that.logger);
          qs('#viewPresetFPS').onclick = () => pcdeditor.command('view_fps').catch(that.logger);

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
          const { instance } = await WebAssembly.instantiateStreaming(fetch('pcdeditor.wasm', fetchOpts), go.importObject)
          go.run(instance);
        }
      });
    }

    loadPCD (path) {
      const that = this;
      return new Promise((resolve, reject) => {
        fetch(path, fetchOpts).then(resp => {
          if (!resp.ok) {
            reject(new Error(`failed to load map.pcd: ${resp.statusText}`));
            return undefined;
          }
          return resp.blob();
        }).then(blob => {
          return that.pcdeditor.importPCD(blob);
        }).then(() => {
          resolve();
        }).catch(e => {
          reject(e);
        });
      });
    }

    load2D (yamlPath, imgPath) {
      const that = this;
      return new Promise((resolve, reject) => {
        fetch(yamlPath, fetchOpts).then(resp => {
          if (!resp.ok) {
            reject(new Error(`failed to load map.yaml: ${resp.statusText}`));
            return undefined;
          }
          return resp.blob();
        }).then(yamlBlob => {
          const img = new global.Image();
          img.onload = async () => {
            await that.pcdeditor.import2D(yamlBlob, img);
            resolve();
          };
          img.error = (e) => {
            reject(new Error(`failed to load map.yaml: ${e.toString()}`));
          };
          img.crossOrigin = 'use-credentials';
          img.src = imgPath;
        }).catch(e => {
          reject(e);
        });
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
      const setupPostmate = () => new Postmate.Model({
        load: async ({ mapPcd, mapYaml, mapPng }) => {
          if (!mapPcd) {
            that.logger('map.pcd is not given');
            return;
          }
          that.loadPCD(mapPcd).catch(that.logger);
          if (!mapYaml) {
            that.logger('map.yaml is not given');
            return;
          }
          if (!mapPng) {
            that.logger('map.png is not given');
            return;
          }
          that.load2D(mapYaml, mapPng).catch(that.logger);
        },
        enableSaveBtn: () => {
          qs('#savePCD').disabled = false
        }
      });
      const parentFrame = await setupPostmate();
      setupControls(that.pcdeditor, parentFrame);
    }
  };
}
