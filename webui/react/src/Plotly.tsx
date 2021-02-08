// Plotly needs to be imported via `require` in order for the `register` method to come through.
/* eslint-disable-next-line @typescript-eslint/no-var-requires */
const Plotly = require('plotly.js/lib/core');

Plotly.register([
  require('plotly.js/lib/parcoords'),
  require('plotly.js/lib/pie'),
  require('plotly.js/lib/scatter'),
]);

export * from 'plotly.js/lib/core';

export default Plotly;
