import React, { useEffect, useState } from 'react';

import NotebookJS from 'notebook';
import { DetError, ErrorType } from 'shared/utils/error';
import handleError from 'utils/error';

import 'vendor/monokai.css';

interface Props {
  file: string;
}

export const parseNotebook = (file: string): string => {
  try {
    const json = JSON.parse(file);
    const notebookJS = NotebookJS.parse(json);
    return notebookJS.render().outerHTML;
  } catch (e) {
    throw new DetError('Unable to parse as Notebook!');
  }
};

const JupyterRenderer: React.FC<Props> = React.memo(({ file }) => {
  const [__html, setHTML] = useState<string>();

  useEffect(() => {
    try {
      const html = parseNotebook(file);
      setHTML(html);
    } catch (error) {
      handleError(error, {
        publicMessage: 'Failed to load selected notebook.',
        publicSubject: 'Unable to parse the selected notebook.',
        silent: true,
        type: ErrorType.Input,
      });
    }
  }, [file]);

  return __html ? (
    <div className="ipynb-renderer-root" dangerouslySetInnerHTML={{ __html }} />
  ) : (
    <div>{file}</div>
  );
});

export default JupyterRenderer;
