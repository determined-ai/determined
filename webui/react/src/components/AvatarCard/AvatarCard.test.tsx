import { render, screen } from '@testing-library/react';

import { DarkLight } from 'utils/themes';

import AvatarCard, { Props } from './AvatarCard';

const setup = (props: Props) => {
  const view = render(<AvatarCard {...props} />);
  return { view };
};

describe('AvatarCard', () => {
  describe('display name', () => {
    it('should display one-word name', () => {
      setup({ darkLight: DarkLight.Light, displayName: 'Admin' });
      expect(screen.getByText('A')).toBeInTheDocument();
      expect(screen.getByText('Admin')).toBeInTheDocument();
    });

    it('should display two-words name', () => {
      setup({ darkLight: DarkLight.Light, displayName: 'Dio Brando' });
      expect(screen.getByText('DB')).toBeInTheDocument();
      expect(screen.getByText('Dio Brando')).toBeInTheDocument();
    });

    it('should display three-words name', () => {
      setup({ darkLight: DarkLight.Light, displayName: 'Gold Experience Requiem' });
      expect(screen.getByText('GR')).toBeInTheDocument();
      expect(screen.getByText('Gold Experience Requiem')).toBeInTheDocument();
    });
  });

  describe('Light Dark Mode', () => {
    it('should be light mode color', () => {
      const { view } = setup({ darkLight: DarkLight.Light, displayName: 'Admin' });
      expect(view.container.querySelector('#avatar')).toHaveStyle(
        'background-color: hsl(290, 63%, 60%)',
      );
    });

    it('should be dark mode color', () => {
      const { view } = setup({ darkLight: DarkLight.Dark, displayName: 'Admin' });
      expect(view.container.querySelector('#avatar')).toHaveStyle(
        'background-color: hsl(290, 63%, 38%)',
      );
    });
  });

  describe('class name', () => {
    it('should not have a base class name', () => {
      const { view } = setup({ darkLight: DarkLight.Light, displayName: 'test' });
      const { container } = view;
      expect(container.children[0]).toHaveAttribute('class');
      expect(container.children[0]).toHaveClass('base');
    });

    it('should have a class name', () => {
      const { view } = setup({
        className: 'test-class',
        darkLight: DarkLight.Light,
        displayName: 'test',
      });
      const { container } = view;
      expect(container.children[0]).toHaveAttribute('class');
      expect(container.children[0]).toHaveClass('base');
      expect(container.children[0]).toHaveClass('test-class');
    });
  });
});
