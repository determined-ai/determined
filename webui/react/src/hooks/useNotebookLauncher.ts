import { useCallback } from 'react';

import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import { openBlank } from 'routes/utils';
import { createNotebook } from 'services/api';
import { commandToTask } from 'utils/types';

interface UseNotebookLauncher {
  launchCpuOnlyNotebook: () => Promise<void>;
  launchNotebook: () => Promise<void>;
}

const useNotebookLauncher = (): UseNotebookLauncher => {
  const launchNotebook = useCallback(async (slots: number) => {
    try {
      const notebook = await createNotebook({ slots });
      const task = commandToTask(notebook);
      if (task.url) openBlank(task.url);
      else throw new Error('Notebook URL not available.');
    } catch (e) {
      handleError({
        error: e,
        level: ErrorLevel.Error,
        message: e.message,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to Launch Notebook',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, []);

  return {
    launchCpuOnlyNotebook: () => launchNotebook(0),
    launchNotebook: () => launchNotebook(1),
  };
};

export default useNotebookLauncher;
