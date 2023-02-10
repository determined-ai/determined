import {
  message as antdMessage,
  Modal as antdModal,
  notification as antdNotification,
  App,
} from 'antd';
import { useAppProps } from 'antd/es/app/context';
import { useEffect } from 'react';

/**
 * Wrapper for static dialog functionality from antd. Regular static instances
 * are not responsive to the theming context, and will appear with the default
 * styling, so we use the app context from antd which hooks into the context.
 * This requires our code to call the `App.useApp` hook somewhere, so we do that
 * in the AppView. We fall back to the vanilla static methods so testing
 * functionality isn't broken.
 */

let notification: useAppProps['notification'] = antdNotification;
let modal: useAppProps['modal'] = antdModal;
let message: useAppProps['message'] = antdMessage;

export const useInitApi = () => {
  const api = App.useApp();
  // minimize reassignments
  useEffect(() => {
    ({ notification, modal, message } = api);
  }, [api]);
};

export { notification, modal, message };
