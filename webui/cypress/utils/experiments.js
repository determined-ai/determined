const exec = require('./shell');

const create = (name, configName) => {
  const experimentDir = './../sample-experiments/' + name;
  return exec(`det experiment create ${experimentDir}/${configName}.yaml ${experimentDir}`);
};

module.exports = {
  create,
};
