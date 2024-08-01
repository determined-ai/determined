import { render } from '@testing-library/react';

import { LeafNode, NonLeafNode, TreePath } from './types';

import { absPathToAddress, onAction, parseInput, queryTree } from '.';

vi.mock('hew/Toast', () => ({
  makeToast: vi.fn(),
}));

const handleAction = vi.fn(() => undefined);
const LEAF_NODE = (title?: string): LeafNode => ({
  onAction: handleAction,
  title: title ?? 'leaf node',
});
const NON_LEAF_NODE = (title?: string): NonLeafNode => ({
  options: [LEAF_NODE()],
  title: title ?? 'non-leaf node',
});

const generateTreePath = (length: number = 5) => {
  const treePath: TreePath = [];
  for (let i = 0; i < length; i++) {
    treePath.push(Math.random() > 0.5 ? LEAF_NODE(`${i}`) : NON_LEAF_NODE(`${i}`));
  }
  return treePath;
};

describe('parseInput', () => {
  it('Parsing input', async () => {
    expect(await parseInput('node', NON_LEAF_NODE())).toEqual({
      path: [NON_LEAF_NODE()],
      query: 'node',
    });
  });
});

describe('absPathToAddress', () => {
  it('Absolute Path', () => {
    const path = generateTreePath();
    expect(absPathToAddress(path)).toEqual(['1', '2', '3', '4']);
  });
});

describe('queryTree', () => {
  it('Leaf Node child', async () => {
    const rootNode = NON_LEAF_NODE();
    expect(await queryTree('node', rootNode)).toEqual(rootNode.options);
  });

  it('Non-leaf Node child', async () => {
    const rootNode = { ...NON_LEAF_NODE(), options: [NON_LEAF_NODE()] };
    expect(await queryTree('node', rootNode)).toEqual(rootNode.options);
  });

  it('Multiple children', async () => {
    const rootNode = {
      ...NON_LEAF_NODE(),
      options: [NON_LEAF_NODE(), NON_LEAF_NODE(), LEAF_NODE()],
    };
    expect(await queryTree('node', rootNode)).toEqual(expect.arrayContaining(rootNode.options));
  });
});

describe('onAction', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  const setup = () => {
    const view = render(<input />);
    return { view };
  };
  it('Leaf Node', async () => {
    const { view } = setup();
    const inputEl = view.getByRole('textbox') as HTMLInputElement;
    const node = LEAF_NODE();
    const queryFn = vi.fn(() => undefined);

    await onAction(inputEl, node, queryFn);
    expect(queryFn).not.toHaveBeenCalled();
    expect(handleAction).toHaveBeenCalled();
  });
  it('Non-leaf Node', async () => {
    const { view } = setup();
    const inputEl = view.getByRole('textbox') as HTMLInputElement;
    const node = NON_LEAF_NODE();
    const queryFn = vi.fn(() => undefined);

    await onAction(inputEl, node, queryFn);
    expect(queryFn).toHaveBeenCalled();
  });
});
