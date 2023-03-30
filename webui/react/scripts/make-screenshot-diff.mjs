/**
 * Extracting the visual regression stuff from playwright so we can diff without
 * having to manually generate target snapshots
 */
import fs from 'fs/promises';
import path from 'path';

// wonky import here bc this module isn't declared as an export -- tread lightly.
import { getComparator } from '../node_modules/playwright-core/lib/utils/comparators.js';

const comparator = getComparator('image/png');
const screenshotFolder = path.resolve(process.cwd(), 'screenshots');

const screenshotLabels = (await fs.readdir(screenshotFolder)).slice(0, 2);

// assuming that the files for each theme are the same.
const [oneList, twoList] = await Promise.all(
  screenshotLabels.map((folder) => {
    const dirPath = path.resolve(screenshotFolder, folder, 'light');
    return fs.readdir(dirPath);
  }),
);

const oneFilesOnlySet = new Set(oneList);
const twoFilesOnlySet = new Set(twoList);
const compFiles = [];
for (const file of oneFilesOnlySet) {
  if (twoFilesOnlySet.has(file)) {
    oneFilesOnlySet.delete(file);
    twoFilesOnlySet.delete(file);
    compFiles.push(file);
  }
}

const results = await Promise.all(
  compFiles.map((file) =>
    Promise.all(
      ['dark', 'light'].map(async (theme) => {
        const filePaths = screenshotLabels.map((folder) =>
          path.resolve(screenshotFolder, folder, theme, file),
        );
        const [oneFile, twoFile] = await Promise.all(filePaths.map((f) => fs.readFile(f)));
        const comparisonResult = comparator(oneFile, twoFile, { maxDiffPixels: 1 });
        return {
          component: path.basename(file, '.png'),
          diff: comparisonResult?.diff,
          left: oneFile,
          right: twoFile,
          theme,
        };
      }),
    ),
  ),
);

const makeComparisonImgTag = (buf, tag) => `
<img class="comparison__comparison_${tag}" src="data:image/png;base64,${buf.toString('base64')}">`;

const makeComparison = (comparison, theme) => `
<div class="comparison__comparison comparison__comparison--${theme} ${
  comparison.diff ? 'comparison__comparison--has-diff' : ''
}">
    <div class="comparison__component-name">${comparison.component}</div>
    <div class="comparison__comparison-box">
        ${makeComparisonImgTag(comparison.left, 'left')}
        ${makeComparisonImgTag(comparison.right, 'right')}
        ${(comparison.diff || '') && makeComparisonImgTag(comparison.diff, 'diff')}
    </div>
</div>
`;
const comparisons = results.map((result) => {
  const [darkComparison, lightComparison] = result;
  return `
<div class="comparison">
    ${makeComparison(darkComparison, 'dark')}
    ${makeComparison(lightComparison, 'light')}
</div>
`;
});

const prelude = `
<!DOCTYPE html>
<html>
<head>
<style>
    html {
        font-family: sans-serif;
    }
    .controls {
        display: flex;
        position: sticky;
        background-color: white;
        top: 0;
        padding-top: 16px;
        padding-bottom: 8px;
        z-index: 1;
    }
    .controls label {
        margin-right: 24px;
    }
    #comparisons {
        display: flex;
        flex-wrap: wrap;
    }
    .comparison__comparison {
        margin-left: 25px;
    }
    .comparison__comparison-box {
        position: relative;
    }
    .comparison__comparison_right,
    .comparison__comparison_diff {
        position: absolute;
        top: 0;
        left: 0;
        display: none;
        pointer-events: none;
    }
    .comparison__comparison--dark {
        display: none;
    }
    .comparison__comparison_left:hover ~ .comparison__comparison_right {
        display: inline-block;
    }
    #comparisons.js_show-dark .comparison__comparison--dark {
        display: block;
    }
    #comparisons.js_show-dark .comparison__comparison--light {
        display: none;
    }
    #comparisons.js_show-diff .comparison__comparison_left:hover ~ .comparison__comparison_diff {
        display: inline-block;
    }
    #comparisons.js_show-diff .comparison__comparison_left:hover ~ .comparison__comparison_right {
        display: none;
    }
    #comparisons.js_hide-blanks .comparison__comparison:not(.comparison__comparison--has-diff) {
        display: none;
    }
</style>
</head>
<body>
<div class="controls">
    <label >Dark mode? <input class="js_darkmode-switch" type="checkbox" /></label>
    <label >Show diff on hover? <input class="js_diff-switch" type="checkbox" /></label>
    <label >Show components with differences only? <input class="js_hide-switch" type="checkbox"/></label>
</div>

<div id="comparisons">
`;
const controlScript = `
<script type="text/javascript">
    const comparisons = document.querySelector('#comparisons')
    const darkSwitch = document.querySelector('.js_darkmode-switch')
    const diffSwitch = document.querySelector('.js_diff-switch')
    const hideSwitch = document.querySelector('.js_hide-switch')
    darkSwitch.addEventListener('change', (e) => {
        comparisons.classList.toggle('js_show-dark', e.target.checked)
    })
    diffSwitch.addEventListener('change', (e) => {
        comparisons.classList.toggle('js_show-diff', e.target.checked)
    })
    hideSwitch.addEventListener('change', (e) => {
    comparisons.classList.toggle('js_hide-blanks', e.target.checked)
    })
    comparisons.classList.toggle('js_show-dark', darkSwitch.checked)
    comparisons.classList.toggle('js_show-diff', diffSwitch.checked)
    comparisons.classList.toggle('js_hide-blanks', hideSwitch.checked)
</script>
`;
const htmlOut = prelude + comparisons.join('') + '</div>' + controlScript;

const htmlPath = path.resolve(process.cwd(), 'screenshot-summary.html');
await fs.writeFile(htmlPath, htmlOut);
