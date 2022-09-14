import React from 'react';
import { CellType, IpynbRenderer } from 'react-ipynb-renderer';

import 'react-ipynb-renderer/dist/styles/onedork.css';

export type IpynbInterface = {
  cells: CellType[];
  nbformat: 3 | 4 | 5;
  worksheets?: {
    cells: CellType[];
  }[];
}

interface Props {
  file: IpynbInterface;
}

const JupyterRenderer: React.FC<Props> = React.memo(({ file }) => {
  return (
    <IpynbRenderer
      bgTransparent={true}
      formulaOptions={{
        // katex by default
        katex: {
          delimiters: 'gitlab', // dollars by default
          katexOptions: { fleqn: false },
        },
        // optional
        renderer: 'mathjax',
      }}
      ipynb={file}
      language="python"
      syntaxTheme="xonokai"
    />
  );
});

export default JupyterRenderer;
