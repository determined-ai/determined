import NotebookJS from 'notebookjs';
import React from 'react';

import './onedork.css';

interface Props {
  file: string;
}

const JupyterRenderer: React.FC<Props> = React.memo(({ file }) => {
  return (
    <div
      className="ipynb-renderer-root"
      dangerouslySetInnerHTML={{
        __html: file && NotebookJS.parse(JSON.parse(file)).render().outerHTML,
      }}
    />
  );
});

export default JupyterRenderer;
