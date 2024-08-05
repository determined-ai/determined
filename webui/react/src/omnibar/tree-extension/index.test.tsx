import { render } from '@testing-library/react';

import dev from './trees/dev';
import { LeafNode, NonLeafNode, TreePath } from './types';

import { absPathToAddress, extension, onAction, parseInput, queryTree } from '.';

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
    treePath.push(i % 2 === 0 ? LEAF_NODE(`${i}`) : NON_LEAF_NODE(`${i}`));
  }
  return treePath;
};

describe('parseInput', () => {
  it('should parse input', async () => {
    expect(await parseInput('node', NON_LEAF_NODE())).toEqual({
      path: [NON_LEAF_NODE()],
      query: 'node',
    });
  });
  it('throws on bad input', async () => {
    await expect(parseInput('this is a bad input', NON_LEAF_NODE())).rejects.toThrow('bad path');
  });
});

describe('absPathToAddress', () => {
  it('should return an absolute path', () => {
    const path = generateTreePath();
    expect(absPathToAddress(path)).toEqual(['1', '2', '3', '4']);
  });
  it('should work with real `dev` tree', () => {
    expect(absPathToAddress(dev)).toEqual(['resetLocalStorage', 'resetUserPreferences']);
  });
});

describe('queryTree', () => {
  it('should return Leaf Node child', async () => {
    const rootNode = NON_LEAF_NODE();
    expect(await queryTree('node', rootNode)).toEqual(rootNode.options);
  });

  it('should return Non-leaf Node child', async () => {
    const rootNode = { ...NON_LEAF_NODE(), options: [NON_LEAF_NODE()] };
    expect(await queryTree('node', rootNode)).toEqual(rootNode.options);
  });

  it('should return multiple children', async () => {
    const rootNode = {
      ...NON_LEAF_NODE(),
      options: [NON_LEAF_NODE(), NON_LEAF_NODE(), LEAF_NODE()],
    };
    expect(await queryTree('node', rootNode)).toEqual(expect.arrayContaining(rootNode.options));
  });
});

describe('extension', () => {
  it('should accept valid input', async () => {
    expect(await extension('dev serverAddress')).toEqual(
      dev.filter((node) => node.title === 'serverAddress'),
    );
  });

  it('should handle errors', async () => {
    expect(await extension('bad input')).toEqual([]);
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
  it('should call handleAction when passed Leaf Node', async () => {
    const { view } = setup();
    const inputEl = view.getByRole('textbox') as HTMLInputElement;
    const node = LEAF_NODE();
    const queryFn = vi.fn(() => undefined);

    await onAction(inputEl, node, queryFn);
    expect(queryFn).not.toHaveBeenCalled();
    expect(handleAction).toHaveBeenCalled();
  });
  it('should call queryFn when passed Non-leaf Node', async () => {
    const { view } = setup();
    const inputEl = view.getByRole('textbox') as HTMLInputElement;
    const node = NON_LEAF_NODE();
    const queryFn = vi.fn(() => undefined);

    await onAction(inputEl, node, queryFn);
    expect(queryFn).toHaveBeenCalled();
  });
});
