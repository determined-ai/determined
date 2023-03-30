/**
 * Extracting the visual regression stuff from playwright so we can diff without
 * having to manually generate target snapshots
 */
import fs from 'fs/promises';
import path from 'path';

// wonky import here bc this module isn't declared as an export -- tread lightly.
import { getComparator } from '../node_modules/playwright-core/lib/utils/comparators.js';

const themes = ['dark', 'light'];
const comparator = getComparator('image/png');
const screenshotFolder = path.resolve(process.cwd(), 'screenshots');

const screenshotLabels = (await fs.readdir(screenshotFolder)).slice(0, 2);

// assuming that the files for each theme are the same.
const [headList, masterList] = await Promise.all(
  screenshotLabels.map((folder) => {
    const dirPath = path.resolve(screenshotFolder, folder, 'light');
    return fs.readdir(dirPath);
  }),
);

const headListOnlySet = new Set(headList);
const masterListOnlySet = new Set(masterList);
const compFiles = [];
for (const file of headListOnlySet) {
  if (masterListOnlySet.has(file)) {
    headListOnlySet.delete(file);
    masterListOnlySet.delete(file);
    compFiles.push(file);
  }
}

const results = await Promise.all(
  compFiles.map((file) =>
    Promise.all(
      themes.map(async (theme) => {
        const filePaths = screenshotLabels.map((folder) =>
          path.resolve(screenshotFolder, folder, theme, file),
        );
        const [headFile, masterFile] = await Promise.all(filePaths.map((f) => fs.readFile(f)));
        const comparisonResult = comparator(headFile, masterFile, { maxDiffPixels: 1 });
        return {
          component: path.basename(file, '.png'),
          diff: comparisonResult?.diff,
          head: headFile,
          master: masterFile,
          theme,
        };
      }),
    ),
  ),
);

const newComponents = await Promise.all(
  [...headListOnlySet].map((file) =>
    Promise.all(
      themes.map(async (theme) => {
        const filePath = path.resolve(screenshotFolder, screenshotLabels[0], theme, file);
        const fileBuffer = await fs.readFile(filePath);
        return {
          component: path.basename(file),
          screenshot: fileBuffer,
          theme,
        };
      }),
    ),
  ),
);

const deletedComponents = await Promise.all(
  [...masterListOnlySet].map((file) =>
    Promise.all(
      themes.map(async (theme) => {
        const filePath = path.resolve(screenshotFolder, screenshotLabels[1], theme, file);
        const fileBuffer = await fs.readFile(filePath);
        return {
          name: path.basename(file),
          screenshot: fileBuffer,
          theme,
        };
      }),
    ),
  ),
);

const makeComparisonImgTag = (buf, tag) => `
<div class="comparison__comparison-box comparison__comparison-box--${tag}">
  <div class="comparison__comparison_box_tagname">${tag}</div>
  <img src="data:image/png;base64,${buf.toString('base64')}">
</div>
`;

const makeComparison = ({ diff, head, master }, theme) => `
<div class="comparison__comparison comparison__comparison--${theme} ${
  diff ? 'comparison__comparison--has-diff' : ''
}">
  ${makeComparisonImgTag(head, 'head')}
  ${makeComparisonImgTag(master, 'master')}
  ${(diff || '') && makeComparisonImgTag(diff, 'diff')}
</div>
`;
const comparisons = results.map(([darkComparison, lightComparison]) => {
  const componentName = darkComparison.component;
  return `
<div class="comparison" id="${componentName.replace(' ', '_')}_diff">
  <div class="comparison__component-name">${componentName}</div>
  ${makeComparison(darkComparison, 'dark')}
  ${makeComparison(lightComparison, 'light')}
</div>
`;
});

const makeStandaloneComponentThemeContainer = ({ screenshot, theme }) => `
<div class="comparison__comparison comparison__comparison--${theme}">
  ${makeComparisonImgTag(screenshot, '')}
</div>
`;

const makeStandaloneComponent = ([darkComponent, lightComponent]) => {
  const { name } = darkComponent;
  return `
<div class="comparison" id="${name.replace(' ', '_')}_component">
  <div class="comparison__component-name">${name}</div>
  ${makeStandaloneComponentThemeContainer(darkComponent, 'dark')}
  ${makeStandaloneComponentThemeContainer(lightComponent, 'light')}
</div>
`;
};
const newSections = newComponents.map(makeStandaloneComponent);
const deletedSections = deletedComponents.map(makeStandaloneComponent);

const comparisonNavItems = results.map((result) => {
  const componentName = result[0].component;
  const hasDiff = result.some((c) => c.diff);
  return `
<li>
  <a href="#${componentName.replace(' ', '_')}_diff">${componentName}${
    hasDiff ? ' -- Changed' : ''
  }</a>
</li>
`;
});

const standaloneNavItem = ([{ name }]) => `
<li>
  <a href="#${name.replace(' ', '_')}_component">${name}</a>
</li>
`;

const newNavItems = newComponents.map(standaloneNavItem);
const deletedNavItems = deletedComponents.map(standaloneNavItem);

const htmlOut = `
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
    background-color: white;
  }
  .comparison__comparison_right,
  .comparison__comparison_diff {
    position: absolute;
    top: 0;
    left: 0;
    display: none;
    pointer-events: none;
  }
  .comparison__comparison_left:hover ~ .comparison__comparison_right {
    display: inline-block;
  }
  .comparison__comparison--dark {
    display: none;
  }
  main.js_show-dark .comparison__comparison--dark {
    display: block;
  }
  main.js_show-dark .comparison__comparison--light {
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
<nav>
  <ul>
    <li><a href="#comparisons"><em>Comparisons</em></a></li>
    ${comparisonNavItems.join('')}
    <li><a href="#new"><em>New Components</em></a></li>
    ${newNavItems.join('')}
    <li><a href="#deleted"><em>Deleted Components</em></a></li>
    ${deletedNavItems.join('')}
  </ul>
</nav>
<main>
  <div id="comparisons">
    ${comparisons.join('')}
  </div>
  <div id="new">
    ${newSections.join('')}
  </div>
  <div id="deleted">
    ${deletedSections.join('')}
  </div>
</main>
<script type="text/javascript">
  const comparisons = document.querySelector('#comparisons')
  const darkSwitch = document.querySelector('.js_darkmode-switch')
  const diffSwitch = document.querySelector('.js_diff-switch')
  const hideSwitch = document.querySelector('.js_hide-switch')
  diffSwitch.addEventListener('change', (e) => {
      comparisons.classList.toggle('js_show-diff', e.target.checked)
  })
  hideSwitch.addEventListener('change', (e) => {
  comparisons.classList.toggle('js_hide-blanks', e.target.checked)
  })
  comparisons.classList.toggle('js_show-diff', diffSwitch.checked)
  comparisons.classList.toggle('js_hide-blanks', hideSwitch.checked)

  const main = document.querySelector('main')
  darkSwitch.addEventListener('change', (e) => {
      main.classList.toggle('js_show-dark', e.target.checked)
  })
  main.classList.toggle('js_show-dark', darkSwitch.checked)
</script>
`;

const htmlPath = path.resolve(process.cwd(), 'screenshot-summary.html');
await fs.writeFile(htmlPath, htmlOut);
