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
const [headList, mainList] = await Promise.all(
  screenshotLabels.map((folder) => {
    const dirPath = path.resolve(screenshotFolder, folder, 'light');
    return fs.readdir(dirPath);
  }),
);

const headListOnlySet = new Set(headList);
const mainListOnlySet = new Set(mainList);
const compFiles = [];
for (const file of headListOnlySet) {
  if (mainListOnlySet.has(file)) {
    headListOnlySet.delete(file);
    mainListOnlySet.delete(file);
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
        const [headFile, mainFile] = await Promise.all(filePaths.map((f) => fs.readFile(f)));
        const comparisonResult = comparator(headFile, mainFile, { maxDiffPixels: 1 });
        return {
          component: path.basename(file, '.png'),
          diff: comparisonResult?.diff,
          head: headFile,
          main: mainFile,
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
          name: path.basename(file, '.png'),
          screenshot: fileBuffer,
          theme,
        };
      }),
    ),
  ),
);

const deletedComponents = await Promise.all(
  [...mainListOnlySet].map((file) =>
    Promise.all(
      themes.map(async (theme) => {
        const filePath = path.resolve(screenshotFolder, screenshotLabels[1], theme, file);
        const fileBuffer = await fs.readFile(filePath);
        return {
          name: path.basename(file, '.png'),
          screenshot: fileBuffer,
          theme,
        };
      }),
    ),
  ),
);

const filteredResults = results.filter((r) => r.some((t) => !!t.diff));
const hasDiff =
  newComponents.length > 0 || deletedComponents.length > 0 || filteredResults.length > 0;

const diffIndicatorPath = path.resolve(process.cwd(), '.diff-detected');
if (hasDiff) {
  fs.writeFile(diffIndicatorPath, '');
}

const makeComparisonImgTag = (buf, tag) => `
<div class="comparison__image-box comparison__image-box--${tag}">
  <div class="comparison__image-box-tagname">${tag}</div>
  <img src="data:image/png;base64,${buf.toString('base64')}">
</div>
`;

const makeComparison = ({ component, diff, head, main }, theme) => `
<div class="comparison__comparison comparison__comparison--${theme} ${
  diff ? 'comparison__comparison--has-diff' : ''
}">
  <h3 class="comparison__component-name">${component}</h3>
  <div class="comparison__comparison-box">
    ${makeComparisonImgTag(head, 'head')}
    ${makeComparisonImgTag(main, 'main')}
    ${(diff || '') && makeComparisonImgTag(diff, 'diff')}
  </div>
</div>
`;
const comparisons = filteredResults.map(([darkComparison, lightComparison]) => {
  const componentName = darkComparison.component;
  return `
<div class="comparison" id="${componentName.replace(' ', '_')}_diff">
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
  <h3 class="comparison__component-name">${name}</h3>
  ${makeStandaloneComponentThemeContainer(darkComponent, 'dark')}
  ${makeStandaloneComponentThemeContainer(lightComponent, 'light')}
</div>
`;
};
const newSections = newComponents.map(makeStandaloneComponent);
const deletedSections = deletedComponents.map(makeStandaloneComponent);

const comparisonNavItems = filteredResults.map((result) => {
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
    background-color: #F9F9F9;
  }
  .controls {
    display: flex;
    position: sticky;
    background-color: #F9F9F9;
    top: 0;
    padding-top: 16px;
    padding-bottom: 8px;
    z-index: 1;
  }
  .controls label {
    margin-right: 24px;
  }
  #comparisons,
  #deleted,
  #new {
    display: flex;
    flex-wrap: wrap;
  }
  .comparison {
    margin-right: 16px;
    margin-bottom: 16px;
  }
  .comparison__image-box {
    background-color: #F9F9F9;
  }
  .comparison__image-box-tagname {
    font-weight: bold;
    margin-bottom: 8px;
  }
  .comparison__image-box--main,
  .comparison__image-box--diff {
    position: absolute;
    top: 0;
    left: 0;
    display: none;
    pointer-events: none;
  }
  .comparison__image-box--head:hover ~ .comparison__image-box--main {
    display: inline-block;
  }
  .comparison__comparison--dark {
    display: none;
  }
  .comparison__comparison-box {
    position: relative;
  }
  .container {
    display: flex;
    position: relative;
  }
  nav {
    position: sticky;
    top: 3em;
    margin-right: 16px;
  }
  nav ul {
    list-style: none;
  }
  nav ul li {
    margin-bottom: 8px;
  }
  main {
    padding-right: 16px;
  }
  main.js_show-dark .comparison__comparison--dark {
    display: block;
  }
  main.js_show-dark .comparison__comparison--light {
    display: none;
  }
  #comparisons.js_show-diff .comparison__image-box--head:hover ~ .comparison__image-box--diff {
    display: inline-block;
  }
  #comparisons.js_show-diff .comparison__image-box--head:hover ~ .comparison__image-box--main {
    display: none;
  }
</style>
</head>
<body>
<div class="controls">
  <label >Dark mode? <input class="js_darkmode-switch" type="checkbox" /></label>
  <label >Show diff on hover? <input class="js_diff-switch" type="checkbox" /></label>
</div>
<div class="container">
  <div>
    <nav>
      <ul>
        <li><a href="#comparisons"><strong>Comparisons</strong></a></li>
        ${comparisonNavItems.join('')}
        <li><a href="#new"><strong>New Components</strong></a></li>
        ${newNavItems.join('')}
        <li><a href="#deleted"><strong>Deleted Components</strong></a></li>
        ${deletedNavItems.join('')}
      </ul>
    </nav>
  </div>
  <main>
    <h2>Comparisons</h2>
    <div id="comparisons">
      ${comparisons.join('')}
    </div>
    <h2>New Components</h2>
    <div id="new">
      ${newSections.join('')}
    </div>
    <h2>Deleted Components</h2>
    <div id="deleted">
      ${deletedSections.join('')}
    </div>
  </main>
</div>
<script type="text/javascript">
  const comparisons = document.querySelector('#comparisons')
  const darkSwitch = document.querySelector('.js_darkmode-switch')
  const diffSwitch = document.querySelector('.js_diff-switch')
  diffSwitch.addEventListener('change', (e) => {
      comparisons.classList.toggle('js_show-diff', e.target.checked)
  })
  comparisons.classList.toggle('js_show-diff', diffSwitch.checked)

  const main = document.querySelector('main')
  darkSwitch.addEventListener('change', (e) => {
      main.classList.toggle('js_show-dark', e.target.checked)
  })
  main.classList.toggle('js_show-dark', darkSwitch.checked)
</script>
`;

const htmlPath = path.resolve(process.cwd(), 'screenshot-summary.html');
await fs.writeFile(htmlPath, htmlOut);
