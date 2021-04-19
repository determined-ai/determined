import { setServerAddress } from 'dev';
import { globalStorage } from 'globalStorage';
import { alertAction } from 'omnibar/actions';
import { Children, TreeNode } from 'omnibar/AsyncTree';

const dev: TreeNode[] = [
  {
    options: [
      {
        onAction: alertAction(`address: ${globalStorage.serverAddress}`),
        title: 'show',
      },
      {
        onCustomInput: (inp: string): Children => {
          return [ {
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
];

export default dev;
