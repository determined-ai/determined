import { Children, TreeNode } from 'omnibar/AsyncTree';
import { Mode, devControls as rr } from 'recordReplay';

const dev: TreeNode[] = [
  {
    onAction: () => window.localStorage.clear(),
    title: 'resetLocalStorage',
  },
  {
    options: [
      {
        onAction: rr.importApiStorage,
        title: 'importFromFile',
      },
      {
        onAction: rr.importApiStorageClipboard,
        title: 'importFromClipboard',
      },
      {
        onAction: rr.exportApiStorage,
        title: 'export',
      },
      {
        aliases: [ 'disable' ],
        onAction: () => rr.setRRMode('disabled'),
        title: 'stop',
      },
      {
        options: (): Children => {
          const modes: Mode[] = [ 'record', 'replay', 'disabled', 'mixed' ];
          return modes.map(mode => ({
            onAction: () => rr.setRRMode(mode),
            title: mode,

          }));
        },
        title: 'setMode',
      },
      {
        onAction: rr.resetApiStorage,
        title: 'resetStore',
      },
      {
        onAction: (): void => {
          rr.setRRMode('disabled');
          rr.resetApiStorage();
          rr.setRRMode('record');
        },
        title: 'easyStartRecording',
      },
      {
        onAction: (): void => {
          rr.setRRMode('disabled');
          rr.exportApiStorage();
          rr.resetApiStorage();
        },
        title: 'easyStopNExport',
      },
    ],
    title: 'recordNReplay',
  },
];

export default dev;
