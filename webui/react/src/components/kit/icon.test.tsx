import { render } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import Icon, { IconNameArray, IconSizeArray } from './Icon';
import type { Props } from './Icon';

const setup = (props?: Props) => {
  const user = userEvent.setup();
  const view = render(<Icon name="star" {...props} />);
  return { user, view };
};

describe('Icon', () => {
  describe('Size of icon', () => {
    it.each(IconSizeArray)('should display a %s-size icon', (size) => {
      const { view } = setup({ name: 'star', size });
      const firstChild = view.container.firstChild;
      expect(firstChild).toHaveClass(...['base', 'icon-star', size]);
    });
  });

  describe('Name of icon', () => {
    // todo: wanna test pseudo-element `content` value, but cannot find a way to test it
    it.each(IconNameArray)('should display a %s icon', (name) => {
      const { view } = setup({ name });
      const firstChild = view.container.firstChild;
      expect(firstChild).toHaveClass(...['base', `icon-${name}`, 'medium']);
    });
  });

  // TODO: test `title`. cannot display title in test-library probably due to <ToolTip>
  // screen.debug() doesnt show tooltip element somehow

  // describe('Tooltip Title', () => {});
});
