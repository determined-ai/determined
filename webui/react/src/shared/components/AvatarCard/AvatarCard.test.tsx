import { render, screen } from '@testing-library/react';
import React from 'react';

import { DarkLight } from 'shared/themes';

import AvatarCard, { Props } from './AvatarCard';

const setup = (props: Props) => {
  return render(<AvatarCard {...props} />);
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
      setup({ darkLight: DarkLight.Light, displayName: 'Admin' });
      expect(screen.getByTestId('avatar-element'))
        .toHaveStyle('background-color: hsl(290, 63%, 60%)');
    });

    it('should be dark mode color', () => {
      setup({ darkLight: DarkLight.Dark, displayName: 'Admin' });
      expect(screen.getByTestId('avatar-element'))
        .toHaveStyle('background-color: hsl(290, 63%, 38%)');
    });
  });

  describe('class name', () => {
    it('should not have a base class name', () => {
      const { container } = setup({ darkLight: DarkLight.Light, displayName: 'test' });
      expect(container.children[0]).toHaveAttribute('class');
      expect(container.children[0]).toHaveClass('base');
    });

    it('should have a class name', () => {
      const { container } = setup(
        { className: 'test-class', darkLight: DarkLight.Light, displayName: 'test' },
      );
      expect(container.children[0]).toHaveAttribute('class');
      expect(container.children[0]).toHaveClass('test-class');
    });

  });
});
