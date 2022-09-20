import { notification as antdNotification } from 'antd';
import { ArgsProps, NotificationApi } from 'antd/lib/notification';

import { telemetryInstance } from 'hooks/useTelemetry';
import { paths } from 'routes/utils';
import history from 'shared/routes/history';
import {
  DetError,
  DetErrorOptions,
  ERROR_NAMESPACE,
  ErrorLevel,
  isDetError,
} from 'shared/utils/error';
import { LoggerInterface } from 'shared/utils/Logger';
import { filterOutLoginLocation } from 'shared/utils/routes';
import { isAborted, isAuthFailure } from 'shared/utils/service';
import { listToStr } from 'shared/utils/string';

const errorLevelMap = {
  [ErrorLevel.Error]: 'error',
  [ErrorLevel.Fatal]: 'error',
  [ErrorLevel.Warn]: 'warn',
};

const openNotification = (e: DetError) => {
  const key = errorLevelMap[e.level] as keyof NotificationApi;
  const notification = antdNotification[key] as (args: ArgsProps) => void;

  notification?.({
    description: e.publicMessage || '',
    message: e.publicSubject || listToStr([e.type, e.level]),
  });
};

const log = (e: DetError) => {
  const key = errorLevelMap[e.level] as keyof LoggerInterface;
  const message = listToStr([`${e.type}:`, e.publicMessage, e.message]);
  e.logger[key](message);
  e.logger[key](e);
};

// Handle an error at the point that you'd want to stop bubbling it up. Avoid handling
// and re-throwing.
const handleError = (error: DetError | unknown, options?: DetErrorOptions): DetError | void => {
  // Ignore request cancellation errors.
  if (isAborted(error)) return;

  let e: DetError | undefined;
  if (isDetError(error)) {
    e = error;
    if (options) e.loadOptions(options);
  } else {
    e = new DetError(error, options);
  }

  if (e.isHandled) {
    if (process.env.IS_DEV) {
      console.warn(`Error "${e.message}" is handled twice.`);
    }
    return;
  }
  e.isHandled = true;

  // Redirect to logout if Auth failure detected (auth token is no longer valid).`
  if (isAuthFailure(e)) {
    // This check accounts for requests that had not been aborted properly prior
    // to the page dismount and end up throwing after the user is logged out.
    const path = window.location.pathname;
    if (!path.includes(paths.login()) && !path.includes(paths.logout())) {
      history.push(paths.logout(), { loginRedirect: filterOutLoginLocation(window.location) });
    }
  }

  // TODO add support for checking, saving, and dismissing class of errors as a user preference
  // using id.
  const skipNotification = e.silent || (e.level === ErrorLevel.Warn && !e.publicMessage);
  if (!skipNotification) openNotification(e);

  // TODO generate stack trace if error is missing? http://www.stacktracejs.com/

  log(e);

  // TODO SEP handle transient failures? eg only take action IF.. (requires keeping state)

  // Report to segment.
  telemetryInstance.track(`${ERROR_NAMESPACE}: ${e.level}`, e);

  // TODO SEP capture a screenshot or more context (generate a call stack)?
  // https://stackblitz.com/edit/react-screen-capture?file=index.js

  return e;
};

export default handleError;
