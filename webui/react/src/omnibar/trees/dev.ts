import { setServerAddress } from 'dev';
import { globalStorage } from 'globalStorage';
import { userPreferencesStorage } from 'hooks/useStorage';
import { alertAction } from 'omnibar/trees/actions';
import { Children, TreeNode } from 'omnibar/types';
import { serverAddress } from 'routes/utils';

const dev: TreeNode[] = [
  {
    options: [
      {
        onAction: () => alertAction(`address: ${serverAddress()}`)() ,
        title: 'show',
      },
      {
        onCustomInput: (inp: string): Children => {
          return [ {
            closeBar: true,
            onAction: () => setServerAddress(inp),
            title: 'Ok',
          } ];
        },
        title: 'set',
      },
      {
        onAction: () => globalStorage.removeServerAddress(),
        title: 'reset',
      },
    ],
    title: 'serverAddress',
  },
  {
    onAction: () => window.localStorage.clear(),
    title: 'resetLocalStorage',
  },
  {
    onAction: () => userPreferencesStorage.reset(),
    title: 'resetUserPreferences',
  },
];

export default dev;
