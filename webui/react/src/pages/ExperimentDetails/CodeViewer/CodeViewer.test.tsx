/* eslint-disable @typescript-eslint/no-unused-vars */
/* eslint-disable max-len */

import { findAllByText, screen, waitFor } from '@testing-library/dom';
import { render } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React, { useEffect } from 'react';
import { act } from 'react-dom/test-utils';
import { unstable_HistoryRouter as HistoryRouter } from 'react-router-dom';

import { SettingsProvider } from 'hooks/useSettingsProvider';
import { paths } from 'routes/utils';
import { StoreProvider as UIProvider } from 'shared/contexts/stores/UI';
import history from 'shared/routes/history';
import { setAuth, setAuthChecked } from 'stores/auth';
import usersStore from 'stores/users';

import CodeViewer, { Props } from './CodeViewer';

const MonacoEditorMock: React.FC = () => <></>;
const hashedFileMock =
  'ZGVzY3JpcHRpb246IG5vb3Bfc2luZ2xlCmNoZWNrcG9pbnRfc3RvcmFnZToKICB0eXBlOiBzaGFyZWRfZnMKICBob3N0X3BhdGg6IC90bXAKICBzdG9yYWdlX3BhdGg6IGRldGVybWluZWQtaW50ZWdyYXRpb24tY2hlY2twb2ludHMKICBzYXZlX3RyaWFsX2Jlc3Q6IDMwCmh5cGVycGFyYW1ldGVyczoKICBnbG9iYWxfYmF0Y2hfc2l6ZTogMzIKICBtZXRyaWNzX3Byb2dyZXNzaW9uOiBkZWNyZWFzaW5nCiAgbWV0cmljc19iYXNlOiAwLjkKICBtZXRyaWNzX3NpZ21hOiAwCnNlYXJjaGVyOgogIG1ldHJpYzogdmFsaWRhdGlvbl9lcnJvcgogIHNtYWxsZXJfaXNfYmV0dGVyOiB0cnVlCiAgbmFtZTogc2luZ2xlCiAgbWF4X2xlbmd0aDoKICAgIHJlY29yZHM6IDgwMDAKcmVwcm9kdWNpYmlsaXR5OgogIGV4cGVyaW1lbnRfc2VlZDogOTk5Cm1pbl92YWxpZGF0aW9uX3BlcmlvZDoKICByZWNvcmRzOiA0MDAwCm1heF9yZXN0YXJ0czogMAplbnRyeXBvaW50OiBtb2RlbF9kZWY6Tm9PcFRyaWFsCg==';

jest.mock('routes/utils', () => {
  return {
    __esModule: true,
    handlePath: () => Promise.resolve(),
    paths: { experimentFileFromTree: () => '/fakePath' },
    serverAddress: () => '',
  };
});

jest.mock('services/api', () => {
  return {
    __esModule: true,
    // encoded file taken from the API
    getExperimentFileFromTree: (id: number) => Promise.resolve(hashedFileMock),
    getExperimentFileTree: (id: number) =>
      Promise.resolve([
        {
          contentLength: 505,
          contentType: 'text/plain; charset=utf-8',
          files: [],
          isDir: false,
          modifiedTime: '2022-01-04T00:58:09Z',
          name: 'single-in-records.yaml',
          path: 'single-in-records.yaml',
        },
        {
          contentLength: 560,
          contentType: 'text/plain; charset=utf-8',
          files: [],
          isDir: false,
          modifiedTime: '2022-01-04T00:58:09Z',
          name: 'single-one-short-step.yaml',
          path: 'single-one-short-step.yaml',
        },
        {
          contentLength: 488,
          contentType: 'text/plain; charset=utf-8',
          files: [],
          isDir: false,
          modifiedTime: '2022-01-04T00:58:09Z',
          name: 'adaptive.yaml',
          path: 'adaptive.yaml',
        },
        {
          contentLength: 10710,
          contentType: 'text/plain; charset=utf-8',
          files: [],
          isDir: false,
          modifiedTime: '2022-06-21T20:30:06Z',
          name: 'model_def.py',
          path: 'model_def.py',
        },
      ]),
    getUserSetting: () => Promise.resolve({ settings: [] }),
  };
});

jest.mock('components/MonacoEditor', () => ({
  __esModule: true,
  default: () => MonacoEditorMock,
}));

jest.mock('hooks/useSettings', () => {
  const actualModule = jest.requireActual('hooks/useSettings');
  const useSettings = jest.fn(() => {
    const settings = { filePath: 'single-in-records.yaml' };
    const updateSettings = jest.fn();

    return { isLoading: false, settings, updateSettings };
  });

  return {
    __esModule: true,
    ...actualModule,
    default: useSettings,
  };
});

global.URL.createObjectURL = jest.fn();
const experimentIdMock = 123;
const user = userEvent.setup();

const Container: React.FC<Props> = (props) => {
  useEffect(() => {
    setAuth({ isAuthenticated: true });
    setAuthChecked();
    usersStore.updateCurrentUser(1);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <SettingsProvider>
      <CodeViewer experimentId={props.experimentId} submittedConfig={props.submittedConfig} />
    </SettingsProvider>
  );
};

const setup = (
  props: Props = { experimentId: experimentIdMock, submittedConfig: hashedFileMock },
) => {
  render(
    <HistoryRouter history={history}>
      <UIProvider>
        <Container {...props} />
      </UIProvider>
    </HistoryRouter>,
  );
};

const getElements = async () => {
  const tree = await screen.findByTestId('fileTree');
  const treeNodes = await findAllByText(tree, /[a-zA-Z\-_]{1,}\./);

  return { treeNodes };
};

describe('CodeViewer', () => {
  afterAll(() => jest.clearAllMocks());

  it('should handle the initial render properly', async () => {
    setup();
    const { treeNodes } = await getElements();

    expect(treeNodes).toHaveLength(4);
  });

  it('should handle clicking in the download icon when opening a file from the tree', async () => {
    const pathBuilderSpy = jest.spyOn(paths, 'experimentFileFromTree').mockReturnValueOnce('');
    setup();

    const { treeNodes } = await getElements();

    await act(() => user.click(treeNodes[1]));

    const button = await screen.findByLabelText('download');

    await act(() => user.click(button));

    waitFor(() => expect(pathBuilderSpy).toHaveBeenCalledWith(123, 'model_def.py'));
  });
});
