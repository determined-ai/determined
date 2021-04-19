import { Children, TreeNode } from 'omnibar/AsyncTree';
import { Mode, requestTextFileUpload, devControls as rr,
  setupRRStorageAndReplay, userPreferencesStorage } from 'recordReplay';
import { refreshPage } from 'utils/browser';

const dev: TreeNode[] = [
  {
    onAction: () => window.localStorage.clear(),
    title: 'resetLocalStorage',
  },
  {
    onAction: () => userPreferencesStorage.reset(),
    title: 'resetUserPreferences',
  },
  {
    options: [
      {
        onAction: (): void => {
          rr.setRRMode('disabled');
          rr.resetApiStorage();
          rr.setRRMode('record');
          // refresh the page to capture the requests that get triggered on initial page load.
          refreshPage();
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
      {
        onAction: async (): Promise<void> => {
          const contents = await requestTextFileUpload();
          setupRRStorageAndReplay(contents);
        },
        title: 'easyImportNReplay',
      },
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

    ],
    title: 'recordNReplay',
  },
];

export default dev;
