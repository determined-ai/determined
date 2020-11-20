const exec = require('child_process').exec;

/**
 * Executes a shell command and return it as a Promise.
 * @param cmd {string}
 * @return {Promise<string>}
 */
function execShellCommand(cmd) {
  return new Promise((resolve, reject) => {
    exec(cmd, (error, stdout, stderr) => {
      if (error) {
        console.warn(error);
        reject(stderr);
      }
      console.log(stdout);
      resolve(stdout);
    });
  });
}

module.exports = execShellCommand;
