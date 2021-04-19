import { setServerAddress } from 'dev';
import { globalStorage } from 'globalStorage';
import { alertAction } from 'omnibar/actions';
import { Children, TreeNode } from 'omnibar/AsyncTree';
import devExtension from 'omnibar/nodes/dev.rr.tmp';
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
  ...devExtension,
];

export default dev;
