import { resetServerAddress, setServerAddress } from 'dev';
import { userPreferencesStorage } from 'hooks/useStorage';
import { alertAction } from 'omnibar/tree-extension/trees/actions';
import { Children, TreeNode } from 'omnibar/tree-extension/types';
import { checkServerAlive, serverAddress } from 'routes/utils';
import { DetError, ErrorLevel, ErrorType } from 'utils/error';

const dev: TreeNode[] = [
  {
    options: [
      {
        onAction: alertAction(`address: ${serverAddress()}`),
        title: 'show',
      },
      {
        label: 'set <URL>',
        onCustomInput: (inp: string): Children => {
          return [
            {
              closeBar: true,
              label: inp || '<URL>',
              onAction: async () => {
                const isAlive = await checkServerAlive(inp);
                if (isAlive) {
                  setServerAddress(inp);
                } else {
                  const error = new DetError(undefined, {
                    isUserTriggered: true,
                    level: ErrorLevel.Error,
                    publicMessage: `Could not find a valid server at "${inp}"`,
                    publicSubject: 'Server not found',
                    type: ErrorType.Ui,
                  });
                  throw error;
                }
              },
              title: inp,
            },
          ];
        },
        title: 'set',
      },
      {
        closeBar: true,
        onAction: () => resetServerAddress(),
        title: 'reset',
      },
    ],
    title: 'serverAddress',
  },
  {
    closeBar: true,
    onAction: () => window.localStorage.clear(),
    title: 'resetLocalStorage',
  },
  {
    closeBar: true,
    onAction: (): void => {
      const resetStorage = userPreferencesStorage();
      resetStorage();
    },
    title: 'resetUserPreferences',
  },
];

export default dev;
