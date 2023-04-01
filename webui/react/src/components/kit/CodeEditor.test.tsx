import { findAllByText, screen, waitFor } from '@testing-library/dom';
import { render } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React, { useEffect } from 'react';
import { BrowserRouter } from 'react-router-dom';

import { SettingsProvider } from 'hooks/useSettingsProvider';
import { paths } from 'routes/utils';
import { StoreProvider as UIProvider } from 'shared/contexts/stores/UI';
import authStore from 'stores/auth';
import userStore from 'stores/users';
import { DetailedUser } from 'types';

import CodeEditor, { Props } from './CodeEditor';

const CURRENT_USER: DetailedUser = { id: 1, isActive: true, isAdmin: false, username: 'bunny' };

const hashedFileMock =
  'ZGVzY3JpcHRpb246IG5vb3Bfc2luZ2xlCmNoZWNrcG9pbnRfc3RvcmFnZToKICB0eXBlOiBzaGFyZWRfZnMKICBob3N0X3BhdGg6IC90bXAKICBzdG9yYWdlX3BhdGg6IGRldGVybWluZWQtaW50ZWdyYXRpb24tY2hlY2twb2ludHMKICBzYXZlX3RyaWFsX2Jlc3Q6IDMwCmh5cGVycGFyYW1ldGVyczoKICBnbG9iYWxfYmF0Y2hfc2l6ZTogMzIKICBtZXRyaWNzX3Byb2dyZXNzaW9uOiBkZWNyZWFzaW5nCiAgbWV0cmljc19iYXNlOiAwLjkKICBtZXRyaWNzX3NpZ21hOiAwCnNlYXJjaGVyOgogIG1ldHJpYzogdmFsaWRhdGlvbl9lcnJvcgogIHNtYWxsZXJfaXNfYmV0dGVyOiB0cnVlCiAgbmFtZTogc2luZ2xlCiAgbWF4X2xlbmd0aDoKICAgIHJlY29yZHM6IDgwMDAKcmVwcm9kdWNpYmlsaXR5OgogIGV4cGVyaW1lbnRfc2VlZDogOTk5Cm1pbl92YWxpZGF0aW9uX3BlcmlvZDoKICByZWNvcmRzOiA0MDAwCm1heF9yZXN0YXJ0czogMAplbnRyeXBvaW50OiBtb2RlbF9kZWY6Tm9PcFRyaWFsCg==';

vi.mock('routes/utils', () => ({
  __esModule: true,
  handlePath: () => Promise.resolve(),
  paths: { experimentFileFromTree: vi.fn().mockReturnValue('/fakePath') },
  serverAddress: () => '',
}));

vi.mock('services/api', () => ({
  // encoded file taken from the API
  getExperimentFileFromTree: () => Promise.resolve(hashedFileMock),
  getExperimentFileTree: () =>
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
}));

vi.mock('components/MonacoEditor', () => ({
  __esModule: true,
  default: () => <></>,
}));

vi.mock('hooks/useSettings', async (importOriginal) => {
  const useSettings = vi.fn(() => {
    const settings = { filePath: 'single-in-records.yaml' };
    const updateSettings = vi.fn();

    return { isLoading: false, settings, updateSettings };
  });

  return {
    __esModule: true,
    ...(await importOriginal<typeof import('hooks/useSettings')>()),
    useSettings,
  };
});

global.URL.createObjectURL = vi.fn();
const experimentIdMock = 123;
const user = userEvent.setup();

const Container: React.FC<Props> = (props) => {
  useEffect(() => {
    authStore.setAuth({ isAuthenticated: true });
    authStore.setAuthChecked();
    userStore.updateCurrentUser(CURRENT_USER);
  }, []);

  return (
    <SettingsProvider>
      <CodeEditor experimentId={props.experimentId} submittedConfig={props.submittedConfig} />
    </SettingsProvider>
  );
};

const setup = (
  props: Props = { experimentId: experimentIdMock, submittedConfig: hashedFileMock },
) => {
  render(
    <BrowserRouter>
      <UIProvider>
        <Container {...props} />
      </UIProvider>
    </BrowserRouter>,
  );
};

const getElements = async () => {
  const tree = await screen.findByTestId('fileTree');
  const treeNodes = await findAllByText(tree, /[a-zA-Z\-_]{1,}\./);

  return { treeNodes };
};

describe('CodeEditor', () => {
  it('should handle the initial render properly', async () => {
    setup();
    const { treeNodes } = await getElements();

    expect(treeNodes).toHaveLength(4);
  });

  it('should handle clicking in the download icon when opening a file from the tree', async () => {
    setup();

    const { treeNodes } = await getElements();

    await user.click(treeNodes[1]);

    const button = await screen.findByLabelText('download');

    await user.click(button);

    await waitFor(() =>
      expect(vi.mocked(paths.experimentFileFromTree)).toHaveBeenCalledWith(
        123,
        'single-in-records.yaml',
      ),
    );
  });
});
