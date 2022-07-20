/* eslint-disable max-len */
/* eslint-disable no-console */
const mockGetExperimentFileFromTree = jest.fn((id: number) => {
  console.log(`getExperimentFileFromTree api called with ${id}`);

  return { file: 'ZGVzY3JpcHRpb246IG5vb3Bfc2luZ2xlCmNoZWNrcG9pbnRfc3RvcmFnZToKICB0eXBlOiBzaGFyZWRfZnMKICBob3N0X3BhdGg6IC90bXAKICBzdG9yYWdlX3BhdGg6IGRldGVybWluZWQtaW50ZWdyYXRpb24tY2hlY2twb2ludHMKICBzYXZlX3RyaWFsX2Jlc3Q6IDMwCmh5cGVycGFyYW1ldGVyczoKICBnbG9iYWxfYmF0Y2hfc2l6ZTogMzIKICBtZXRyaWNzX3Byb2dyZXNzaW9uOiBkZWNyZWFzaW5nCiAgbWV0cmljc19iYXNlOiAwLjkKICBtZXRyaWNzX3NpZ21hOiAwCnNlYXJjaGVyOgogIG1ldHJpYzogdmFsaWRhdGlvbl9lcnJvcgogIHNtYWxsZXJfaXNfYmV0dGVyOiB0cnVlCiAgbmFtZTogc2luZ2xlCiAgbWF4X2xlbmd0aDoKICAgIHJlY29yZHM6IDgwMDAKcmVwcm9kdWNpYmlsaXR5OgogIGV4cGVyaW1lbnRfc2VlZDogOTk5Cm1pbl92YWxpZGF0aW9uX3BlcmlvZDoKICByZWNvcmRzOiA0MDAwCm1heF9yZXN0YXJ0czogMAplbnRyeXBvaW50OiBtb2RlbF9kZWY6Tm9PcFRyaWFsCg==' };
});

const mockGetExperimentFileTree = jest.fn((id: number) => {
  console.log(`getExperimentFileTree api called with ${id}`);

  return {
    files: [
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
    ],
  };
});

jest.mock('services/api', () => {
  const actualModule = jest.requireActual('services/api');

  return {
    ...actualModule,
    getExperimentFileFromTree: mockGetExperimentFileFromTree,
    getExperimentFileTree: mockGetExperimentFileTree,
  };
});

import { getAllByText, screen } from '@testing-library/dom';
import { render } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';

import CodeViewer, { Props } from './CodeViewer';

const experimentIdMock = 123;
const user = userEvent.setup();

const setup = (props: Props = { experimentId: experimentIdMock }) => {
// const setup = () => {
  // const view = render(<div />);
  const view = render(<CodeViewer experimentId={props.experimentId} />);

  return view;
};

describe('CodeViewer', () => {
  it('should render the file tree and the text editor properly', async () => {
    setup();
    const tree = await screen.findByTestId('fileTree');
    const treeNodes = getAllByText(tree, /[a-zA-Z\-_]{1,}\./g);
    expect(treeNodes).toHaveLength(4);
    expect([ 1, 2, 3, 4 ]).toHaveLength(4);
  });
});
