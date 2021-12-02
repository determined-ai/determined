import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import { launchJupyterLab as apiLaunchJupyterLab } from 'services/api';
import { previewJupyterLab as apiPreviewJupyterLab } from 'services/api';
import { RawJson } from 'types';
import { openCommand } from 'wait';

interface JupyterLabOptions {
  name?: string,
  pool?:string,
  slots?: number,
  templateName?: string,
}

interface JupyterLabLaunchOptions extends JupyterLabOptions {
  config?: RawJson,
}

interface JupyterLabHooks {
  launchJupyterLab: (options: JupyterLabLaunchOptions) => Promise<void>;
  previewJupyterLab: (options: JupyterLabOptions) => Promise<RawJson>;
}

export const launchJupyterLab = async (
  options: JupyterLabLaunchOptions = {},
): Promise<void> => {
  try {
    const jupyterLab = await apiLaunchJupyterLab({
      config: options.config || {
        description: options.name === '' ? undefined : options.name,
        resources: {
          resource_pool: options.pool === '' ? undefined : options.pool,
          slots: options.slots,
        },
      },
      templateName: options.templateName === '' ? undefined : options.templateName,
    });
    openCommand(jupyterLab);
  } catch (e) {
    handleError({
      error: e,
      level: ErrorLevel.Error,
      message: e.message,
      publicMessage: 'Please try again later.',
      publicSubject: 'Unable to Launch JupyterLab',
      silent: false,
      type: ErrorType.Server,
    });
  }
};

export const previewJupyterLab = async (
  options: JupyterLabOptions = {},
): Promise<RawJson> => {
  try {
    const config = await apiPreviewJupyterLab({
      config: {
        description: options.name === '' ? undefined : options.name,
        resources: {
          resource_pool: options.pool === '' ? undefined : options.pool,
          slots: options.slots,
        },
      },
      preview: true,
      templateName: options.templateName === '' ? undefined : options.templateName,
    });
    return config;
  } catch (e) {
    throw new Error('Unable to load JupyterLab config.');
  }
};

const useJupyterLab = (): JupyterLabHooks => {
  return { launchJupyterLab, previewJupyterLab };
};

export default useJupyterLab;
