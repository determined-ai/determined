import { pathToRegexp } from 'path-to-regexp';

import { globalStorage } from 'globalStorage';
import { ClusterApi, Configuration } from 'services/api-ts-sdk';
import { BrandingType } from 'stores/determinedInfo';
import { RouteConfig } from 'types';
import { CommandTask } from 'types';
import {
  AnyMouseEvent,
  AnyMouseEventHandler,
  isAbsolutePath,
  isFullPath,
  isNewTabClickEvent,
  openBlank,
  reactHostAddress,
  routeToExternalUrl,
  routeToReactUrl,
} from 'utils/routes';
import { waitPageUrl } from 'utils/wait';

import routes from './routes';

// serverAddress returns determined cluster (master) address.
export function serverAddress(path = ''): string {
  if (!!path && isFullPath(path)) return path;

  // Prioritize dynamically set address.
  const customServer = globalStorage.serverAddress || (process.env.SERVER_ADDRESS as string);

  return (customServer || reactHostAddress()) + path;
}

// checks to see if the provided address resolves to a live Determined server or not.
export const checkServerAlive = async (address?: string): Promise<boolean> => {
  address = address || serverAddress();
  try {
    const clusterApi = new ClusterApi(new Configuration({ basePath: address }));
    const data = await clusterApi.getMaster();
    const attrs = ['version', 'masterId', 'clusterId'];
    for (const attr of attrs) {
      // The server doesn't look like a determined server.
      if (!(attr in data)) return false;
    }
    return true;
  } catch (_) {
    return false;
  }
};

export const linkPath = (aPath: string, external = false): string => {
  if (isFullPath(aPath)) return aPath;
  let path;
  if (external) {
    if (isAbsolutePath(aPath)) {
      path = serverAddress() + aPath;
    } else {
      path = aPath;
    }
  } else {
    path = process.env.PUBLIC_URL + aPath;
  }
  return path;
};

const routeById: Record<string, RouteConfig> = routes.reduce((acc, cur) => {
  acc[cur.id] = cur;
  return acc;
}, {} as Record<string, RouteConfig>);

export const paths = {
  admin: (tab = ''): string => {
    return `/admin/${tab}`;
  },
  clusterLogs: (): string => {
    return '/logs';
  },
  clusters: (): string => {
    return '/clusters';
  },
  dashboard: (): string => {
    return '/dashboard';
  },
  docs: (suffix?: string): string => {
    return `/docs${suffix || ''}`;
  },
  experimentComparison: (experimentIds: string[]): string => {
    return `/experiment-compare?id=${experimentIds.join('&id=')}`;
  },
  experimentDetails: (experimentId: number | string): string => {
    return `/experiments/${experimentId}`;
  },
  experimentFileFromTree: (experimentId: number | string, filePath: string): string => {
    return `/experiments/${experimentId}/file/download?path=${encodeURIComponent(filePath)}`;
  },
  experimentList: (): string => {
    return '/experiments';
  },
  experimentModelDef: (experimentId: number | string): string => {
    return `/experiments/${experimentId}/model_def`;
  },
  interactive: (command: CommandTask, maxSlotsExceeded = false): string => {
    const path = [
      'interactive',
      command.id,
      command.type,
      command.name,
      command.resourcePool,
      waitPageUrl(command),
    ]
      .map(encodeURIComponent)
      .join('/');
    return `/${path}/?currentSlotsExceeded=${maxSlotsExceeded}`;
  },
  jobs: (): string => {
    return routeById.jobs.path;
  },
  login: (): string => {
    return '/login';
  },
  logout: (): string => {
    return '/logout';
  },
  modelDetails: (modelNameOrId: string): string => {
    return `/models/${encodeURIComponent(modelNameOrId)}`;
  },
  modelList: (): string => {
    return '/models';
  },
  modelVersionDetails: (modelNameOrId: string, versionNum: number | string): string => {
    return `/models/${encodeURIComponent(modelNameOrId)}/versions/${versionNum}`;
  },
  projectDetails: (projectId: number | string): string => {
    return `/projects/${projectId}/experiments`;
  },
  projectDetailsBasePath: (projectId: number | string): string => {
    return `/projects/${projectId}`;
  },
  reload: (path: string): string => {
    return `/reload?${new URLSearchParams({ path })}`;
  },
  resourcePool: (name: string): string => {
    return `/resourcepool/${name}`;
  },
  settings: (tab = ''): string => {
    return `/settings/${tab}`;
  },
  submitProductFeedback: (branding: BrandingType): string => {
    return branding === BrandingType.Determined
      ? 'https://airtable.com/shr87rnMuHhiDTpLo'
      : 'https://airtable.com/shrodYROolF0E1iYf';
  },
  taskList: (): string => {
    return '/tasks';
  },
  taskLogs: (task: Pick<CommandTask, 'id' | 'name' | 'type'>): string => {
    return `/${task.type}/${task.id}/logs?id=${task.name}`;
  },
  trialDetails: (trialId: number | string, experimentId?: number | string): string => {
    if (!experimentId) {
      return `/trials/${trialId}`;
    }
    return `/experiments/${experimentId}/trials/${trialId}`;
  },
  trialLogs: (trialId: number | string, experimentId: number | string): string => {
    return `/experiments/${experimentId}/trials/${trialId}/logs`;
  },
  uncategorized: (): string => {
    return '/projects/1/experiments';
  },
  users: (): string => {
    return '/users';
  },
  webhooks: (): string => {
    return '/webhooks';
  },
  workspaceDetails: (workspaceId: number | string, tab = 'projects'): string => {
    return `/workspaces/${workspaceId}/${tab}`;
  },
  workspaceList: (): string => {
    return '/workspaces';
  },
};
/*
  routeAll determines whether a path should be routed through internal React router or hanled
  by the browser.
  input `path` should include the PUBLIC_URL if there is one set. eg if react is being served
  in a subdirectory.
*/
export const routeAll = (path: string): void => {
  const matchingReactRoute = findReactRoute(path);
  if (!matchingReactRoute) {
    routeToExternalUrl(path);
  } else {
    routeToReactUrl(path);
  }
};
export const handlePath = (
  event: AnyMouseEvent,
  options: {
    external?: boolean;
    onClick?: AnyMouseEventHandler;
    path?: string;
    popout?: boolean | 'tab' | 'window';
  } = {},
): void => {
  event.preventDefault();

  const href = options.path ? linkPath(options.path, options.external) : undefined;

  if (options.onClick) {
    options.onClick(event);
  } else if (href) {
    if (isNewTabClickEvent(event) || options.popout) {
      /**
       * `location=0` forces a new window instead of a tab to open.
       * https://stackoverflow.com/questions/726761/javascript-open-in-a-new-window-not-tab
       */
      const windowFeatures = options.popout === 'window' ? 'location=0' : undefined;
      openBlank(href, undefined, windowFeatures);
    } else {
      routeAll(href);
    }
  }
};
export const findReactRoute = (url: string): RouteConfig | undefined => {
  if (isFullPath(url)) {
    if (!url.startsWith(reactHostAddress())) return undefined;
    // Fit it into a relative path
    url = url.replace(reactHostAddress(), '');
  }
  if (!url.startsWith(process.env.PUBLIC_URL)) {
    return undefined;
  }
  // Check to see if the path matches any of the defined app routes.
  const pathname = url.replace(process.env.PUBLIC_URL, '');
  return routes
    .filter((route) => route.path !== '*')
    .find((route) => {
      const routeRegex = pathToRegexp(route.path);
      return routeRegex.test(pathname);
    });
};
