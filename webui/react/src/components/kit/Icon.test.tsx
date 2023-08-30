import { render } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import Icon, { IconNameArray, IconSizeArray, type Props, svgIcons } from 'components/kit/Icon';

const setup = (props?: Props) => {
  const user = userEvent.setup();
  const props_: Partial<Props> = props ?? {};
  const title = ('title' in props_ ? props_.title : undefined) ?? 'Icon';
  const view = render(
    <Icon
      color={props?.color}
      name={props?.name ?? 'star'}
      showTooltip
      size={props?.size}
      title={title}
    />,
  );
  return { user, view };
};

describe('Icon', () => {
  describe('Size of icon', () => {
    it.each(IconSizeArray)('should display a %s-size icon', (size) => {
      const { view } = setup({ name: 'star', size, title: size });
      const firstChild = view.container.firstChild;
      expect(firstChild).toHaveClass(...['base', 'icon-star', size]);
    });
  });

  describe('Name of icon', () => {
    // todo: wanna test pseudo-element `content` value, but cannot find a way to test it
    it.each(IconNameArray)('should display a %s icon', (name) => {
      const { view } = setup({ name, title: name });
      const firstChild = view.container.firstChild;
      if (!(svgIcons as readonly string[]).includes(name)) {
        expect(firstChild).toHaveClass(...['base', `icon-${name}`, 'medium']);
      } else {
        expect(firstChild?.firstChild?.nodeName).toBe('svg');
      }
    });
  });

  // TODO: test `title`. cannot display title in test-library probably due to <ToolTip>
  // screen.debug() doesnt show tooltip element somehow

  // describe('Tooltip Title', () => {});
});
