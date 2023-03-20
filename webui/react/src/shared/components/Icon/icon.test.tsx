import { render } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import Icon from './Icon';
import type { Props } from './Icon';

const setup = (props?: Props) => {
  const user = userEvent.setup();
  const view = render(<Icon {...props} />);
  return { user, view };
};

describe('Icon', () => {
  it('should display a default icon', () => {
    const { view } = setup();
    const firstChild = view.container.firstChild;
    expect(firstChild).toHaveClass(...['base', 'icon-star', 'medium']);
    expect(firstChild).toHaveStyle({ 'font-size': 'var(--icon-medium)' });
  });

  describe('Size', () => {
    it('should display a tiny-size icon', () => {
      const { view } = setup({ size: 'tiny' });
      const firstChild = view.container.firstChild;
      expect(firstChild).toHaveClass(...['base', 'icon-star', 'tiny']);
      expect(firstChild).toHaveStyle({ 'font-size': 'var(--icon-tiny)' });
    });

    it('should display a small-size icon', () => {
      const { view } = setup({ size: 'small' });
      const firstChild = view.container.firstChild;
      expect(firstChild).toHaveClass(...['base', 'icon-star', 'small']);
      expect(firstChild).toHaveStyle({ 'font-size': 'var(--icon-small)' });
    });

    it('should display a medium-size icon', () => {
      const { view } = setup({ size: 'medium' });
      const firstChild = view.container.firstChild;
      expect(firstChild).toHaveClass(...['base', 'icon-star', 'medium']);
      expect(firstChild).toHaveStyle({ 'font-size': 'var(--icon-medium)' });
    });

    it('should display a large-size icon', () => {
      const { view } = setup({ size: 'large' });
      const firstChild = view.container.firstChild;
      expect(firstChild).toHaveClass(...['base', 'icon-star', 'large']);
      expect(firstChild).toHaveStyle({ 'font-size': 'var(--icon-large)' });
    });

    it('should display a big-size icon', () => {
      const { view } = setup({ size: 'big' });
      const firstChild = view.container.firstChild;
      expect(firstChild).toHaveClass(...['base', 'icon-star', 'big']);
      expect(firstChild).toHaveStyle({ 'font-size': 'var(--icon-big)' });
    });

    it('should display a great-size icon', () => {
      const { view } = setup({ size: 'great' });
      const firstChild = view.container.firstChild;
      expect(firstChild).toHaveClass(...['base', 'icon-star', 'great']);
      expect(firstChild).toHaveStyle({ 'font-size': 'var(--icon-great)' });
    });

    it('should display a huge-size icon', () => {
      const { view } = setup({ size: 'huge' });
      const firstChild = view.container.firstChild;
      expect(firstChild).toHaveClass(...['base', 'icon-star', 'huge']);
      expect(firstChild).toHaveStyle({ 'font-size': 'var(--icon-huge)' });
    });

    it('should display a enormous-size icon', () => {
      const { view } = setup({ size: 'enormous' });
      const firstChild = view.container.firstChild;
      expect(firstChild).toHaveClass(...['base', 'icon-star', 'enormous']);
      expect(firstChild).toHaveStyle({ 'font-size': 'var(--icon-enormous)' });
    });

    it('should display a giant-size icon', () => {
      const { view } = setup({ size: 'giant' });
      const firstChild = view.container.firstChild;
      expect(firstChild).toHaveClass(...['base', 'icon-star', 'giant']);
      expect(firstChild).toHaveStyle({ 'font-size': 'var(--icon-giant)' });
    });

    it('should display a jumbo-size icon', () => {
      const { view } = setup({ size: 'jumbo' });
      const firstChild = view.container.firstChild;
      expect(firstChild).toHaveClass(...['base', 'icon-star', 'jumbo']);
      expect(firstChild).toHaveStyle({ 'font-size': 'var(--icon-jumbo)' });
    });

    it('should display a mega-size icon', () => {
      const { view } = setup({ size: 'mega' });
      const firstChild = view.container.firstChild;
      expect(firstChild).toHaveClass(...['base', 'icon-star', 'mega']);
      expect(firstChild).toHaveStyle({ 'font-size': 'var(--icon-mega)' });
    });
  });

  describe('Name of icon', () => {
    // todo: wanna test pseudo-element `content` value, but cannot find a way to test it
    it('should display a star icon', () => {
      const { view } = setup({ name: 'star' });
      const firstChild = view.container.firstChild;
      expect(firstChild).toHaveClass(...['base', 'icon-star', 'medium']);
    });

    it('should display a tasks icon', () => {
      const { view } = setup({ name: 'tasks' });
      const firstChild = view.container.firstChild;
      expect(firstChild).toHaveClass(...['base', 'icon-tasks', 'medium']);
    });

    it('should display a tensor-board icon', () => {
      const { view } = setup({ name: 'tensor-board' });
      const firstChild = view.container.firstChild;
      expect(firstChild).toHaveClass(...['base', 'icon-tensor-board', 'medium']);
    });

    it('should display a tensorflow icon', () => {
      const { view } = setup({ name: 'tensorflow' });
      const firstChild = view.container.firstChild;
      expect(firstChild).toHaveClass(...['base', 'icon-tensorflow', 'medium']);
    });
  });

  // TODO: test `title`. cannot display title in test-library probably due to <ToolTip>
  // screen.debug() doesnt show tooltip element somehow

  // describe('Tooltip Title', () => {});
});
