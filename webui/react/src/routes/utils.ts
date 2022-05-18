import { pathToRegexp } from 'path-to-regexp';
import queryString from 'query-string';

import { globalStorage } from 'globalStorage';
import { ClusterApi, Configuration } from 'services/api-ts-sdk';
import { BrandingType, CommandTask } from 'types';
import { waitPageUrl } from 'wait';

import { RouteConfig } from '../shared/types';
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
} from '../shared/utils/routes';

import routes from './routes';

// serverAddress returns determined cluster (master) address.
export const serverAddress = (path = ''): string => {
  if (!!path && isFullPath(path)) return path;

  // Prioritize dynamically set address.
  const customServer = globalStorage.serverAddress
    || process.env.SERVER_ADDRESS as string;

  return (customServer || reactHostAddress()) + path;
};

// checks to see if the provided address resolves to a live Determeind server or not.
export const checkServerAlive = async (address?: string): Promise<boolean> => {
  address = address || serverAddress();
  try {
    const clusterApi = new ClusterApi(new Configuration({ basePath: address }));
    const data = await clusterApi.getMaster();
    const attrs = [ 'version', 'masterId', 'clusterId' ];
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
  cluster: (): string => {
    return '/clusters';
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
  experimentDetails: (experimentId: number | string): string => {
    return `/experiments/${experimentId}`;
  },
  experimentList: (): string => {
    return '/experiments';
  },
  experimentModelDef: (experimentId: number | string): string => {
    return `/experiments/${experimentId}/model_def`;
  },
  interactive: (command: CommandTask): string => {
    return `/interactive/${command.id}/${command.type}/
      ${command.name}/${command.resourcePool}/${encodeURIComponent(waitPageUrl(command))}`;
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
  modelDetails: (modelName: string): string => {
    return `/models/${encodeURIComponent(modelName)}`;
  },
  modelList: (): string => {
    return '/models';
  },
  modelVersionDetails: (modelName: string, versionId: number | string): string => {
    return `/models/${encodeURIComponent(modelName)}/versions/${versionId}`;
  },
  reload: (path: string): string => {
    return `/reload?${queryString.stringify({ path })}`;
  },
  resourcePool: (name: string): string => {
    return `/resourcepool/${name}`;
  },
  submitProductFeedback: (branding: BrandingType): string => {
    return branding === BrandingType.Determined
      ? 'https://airtable.com/shr87rnMuHhiDTpLo'
      : 'https://airtable.com/shrodYROolF0E1iYf';
  },
  taskList: (): string => {
    return '/tasks';
  },
  taskLogs: (task: CommandTask): string => {
    return`/${task.type}/${task.id}/logs?id=${task.name}`;
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
  users: (): string => {
    return '/users';
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
    external?: boolean,
    onClick?: AnyMouseEventHandler,
    path?: string,
    popout?: boolean,
  } = {},
): void => {
  // FIXME As of v17, e.persist() doesn’t do anything because the SyntheticEvent is no longer
  // pooled.
  // event.persist();
  event.preventDefault();

  const href = options.path ? linkPath(options.path, options.external) : undefined;

  if (options.onClick) {
    options.onClick(event);
  } else if (href) {
    if (isNewTabClickEvent(event) || options.popout) {
      openBlank(href);
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
    .filter(route => route.path !== '*')
    .find(route => {
      const routeRegex = pathToRegexp(route.path);
      return routeRegex.test(pathname);
    });
};
