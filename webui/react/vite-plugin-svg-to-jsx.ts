import { promises as fs } from 'fs';

import { transform as swcTransform } from '@swc/core';
import { jsx, toJs } from 'estree-util-to-js';
import { fromHtml } from 'hast-util-from-html';
import { toEstree } from 'hast-util-to-estree';
import { optimize, Config as SvgoConfig } from 'svgo';
import { Plugin } from 'vite';

const propsId = {
  name: 'props',
  type: 'Identifier',
} as const;
const ReactComponentId = {
  name: 'ReactComponent',
  type: 'Identifier',
} as const;
export const svgToReact = (config: SvgoConfig): Plugin[] => {
  return [
    {
      enforce: 'pre',
      async load(fullPath) {
        const [filePath, query] = fullPath.split('?', 2);
        // treat the svg as normal if there's a query
        if (filePath.endsWith('.svg') && !query) {
          const svgCode = await fs.readFile(filePath, { encoding: 'utf8' });
          const optimizedSvgCode = optimize(svgCode, config);
          const hast = fromHtml(optimizedSvgCode.data, {
            fragment: true,
            space: 'svg',
          });
          // get the first child of the root node so we don't dump everything into a fragment
          const estree = toEstree(hast.children[0], {
            space: 'svg',
          });
          const expressionStatement = estree.body[0];
          if (expressionStatement.type !== 'ExpressionStatement') {
            throw new Error('Parse error when adding props to jsx');
          }
          const jsxExpression = expressionStatement.expression;
          if (jsxExpression.type !== 'JSXElement') {
            throw new Error('Parse error when adding props to jsx');
          }
          // spread props into the element
          jsxExpression.openingElement.attributes.push({
            argument: propsId,
            type: 'JSXSpreadAttribute',
          });
          estree.body[0] = {
            declaration: {
              declarations: [
                {
                  id: ReactComponentId,
                  init: {
                    body: jsxExpression,
                    expression: true,
                    params: [propsId],
                    type: 'ArrowFunctionExpression',
                  },
                  type: 'VariableDeclarator',
                },
              ],
              kind: 'const',
              type: 'VariableDeclaration',
            },
            specifiers: [],
            type: 'ExportNamedDeclaration',
          };
          estree.body.push({
            declaration: ReactComponentId,
            type: 'ExportDefaultDeclaration',
          });
          const newCode = toJs(estree, { handlers: jsx });
          return swcTransform(newCode.value, {
            filename: filePath,
            jsc: {
              parser: { jsx: true, syntax: 'ecmascript' },
              target: 'es2020',
              transform: {
                react: {
                  runtime: 'automatic',
                },
              },
            },
          });
        }
      },
      name: 'svg-to-react:transform',
    },
  ];
};
