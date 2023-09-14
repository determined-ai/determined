import { render, screen } from '@testing-library/react';

import { DarkLight } from 'components/kit/Theme';

import { ImageAlert, ImageEmpty, ImageWarning, type Props } from './Image';

const setupImageAlert = (props?: Props) => {
  const view = render(<ImageAlert {...props} />);
  return { view };
};

const setupImageEmpty = () => {
  const view = render(<ImageEmpty />);
  return { view };
};

const setupImageWarning = (props?: Props) => {
  const view = render(<ImageWarning {...props} />);
  return { view };
};

describe('Image', () => {
  describe('ImageAlert', () => {
    it('should display ImageAlert with implicit props', () => {
      const { view } = setupImageAlert();
      const svg = view.container.querySelector('svg');
      expect(screen.getByTitle('Alert')).toBeInTheDocument();
      expect(view.container.firstChild).toHaveClass('alert');
      expect(view.container.firstChild).not.toHaveClass('dark');
      expect(view.container.firstChild).toHaveAttribute('fill', 'none');
      expect(view.container.firstChild).toHaveAttribute('height', '100');
      expect(view.container.firstChild).toHaveAttribute('width', '100');
      expect(view.container.firstChild).toHaveAttribute('viewBox', '0 0 1024 1024');
      expect(view.container.firstChild).toHaveAttribute('xmlns', 'http://www.w3.org/2000/svg');
      expect(svg).not.toBeEmptyDOMElement();
    });

    it('should display ImageAlert with explicit props, Light Mode', () => {
      const { view } = setupImageAlert({ darkLight: DarkLight.Light });
      const svg = view.container.querySelector('svg');
      expect(screen.getByTitle('Alert')).toBeInTheDocument();
      expect(view.container.firstChild).toHaveClass('alert');
      expect(view.container.firstChild).not.toHaveClass('dark');
      expect(view.container.firstChild).toHaveAttribute('fill', 'none');
      expect(view.container.firstChild).toHaveAttribute('height', '100');
      expect(view.container.firstChild).toHaveAttribute('width', '100');
      expect(view.container.firstChild).toHaveAttribute('viewBox', '0 0 1024 1024');
      expect(view.container.firstChild).toHaveAttribute('xmlns', 'http://www.w3.org/2000/svg');
      expect(svg).not.toBeEmptyDOMElement();
    });

    it('should display ImageAlert with explicit props, Dark Mode', () => {
      const { view } = setupImageAlert({ darkLight: DarkLight.Dark });
      const svg = view.container.querySelector('svg');
      expect(screen.getByTitle('Alert')).toBeInTheDocument();
      expect(view.container.firstChild).toHaveClass(...['alert', 'dark']);
      expect(view.container.firstChild).toHaveAttribute('fill', 'none');
      expect(view.container.firstChild).toHaveAttribute('height', '100');
      expect(view.container.firstChild).toHaveAttribute('width', '100');
      expect(view.container.firstChild).toHaveAttribute('viewBox', '0 0 1024 1024');
      expect(view.container.firstChild).toHaveAttribute('xmlns', 'http://www.w3.org/2000/svg');
      expect(svg).not.toBeEmptyDOMElement();
    });
  });

  describe('ImageEmpty', () => {
    it('should display ImageEmpty with implicit props', () => {
      const { view } = setupImageEmpty();
      const svg = view.container.querySelector('svg');
      expect(screen.getByTitle('Empty')).toBeInTheDocument();
      expect(view.container.firstChild).toHaveClass(...['ant-empty-img-simple']);
      expect(view.container.firstChild).not.toHaveAttribute('fill', 'none');
      expect(view.container.firstChild).toHaveAttribute('height', '100');
      expect(view.container.firstChild).toHaveAttribute('width', '100');
      expect(view.container.firstChild).toHaveAttribute('viewBox', '-8 -5 80 51');
      expect(view.container.firstChild).toHaveAttribute('xmlns', 'http://www.w3.org/2000/svg');
      expect(svg).not.toBeEmptyDOMElement();
    });
  });

  describe('ImageWarning', () => {
    it('should display ImageWarning with implicit props', () => {
      const { view } = setupImageWarning();
      const svg = view.container.querySelector('svg');
      expect(screen.getByTitle('Warning')).toBeInTheDocument();
      expect(view.container.firstChild).toHaveClass('warning');
      expect(view.container.firstChild).not.toHaveClass('dark');
      expect(view.container.firstChild).toHaveAttribute('fill', 'none');
      expect(view.container.firstChild).toHaveAttribute('height', '100');
      expect(view.container.firstChild).toHaveAttribute('width', '100');
      expect(view.container.firstChild).toHaveAttribute('viewBox', '0 0 1024 1024');
      expect(view.container.firstChild).toHaveAttribute('xmlns', 'http://www.w3.org/2000/svg');
      expect(svg).not.toBeEmptyDOMElement();
    });

    it('should display ImageWarning with explicit props, Light Mode', () => {
      const { view } = setupImageWarning({ darkLight: DarkLight.Light });
      const svg = view.container.querySelector('svg');
      expect(screen.getByTitle('Warning')).toBeInTheDocument();
      expect(view.container.firstChild).toHaveClass('warning');
      expect(view.container.firstChild).not.toHaveClass('dark');
      expect(view.container.firstChild).toHaveAttribute('fill', 'none');
      expect(view.container.firstChild).toHaveAttribute('height', '100');
      expect(view.container.firstChild).toHaveAttribute('width', '100');
      expect(view.container.firstChild).toHaveAttribute('viewBox', '0 0 1024 1024');
      expect(view.container.firstChild).toHaveAttribute('xmlns', 'http://www.w3.org/2000/svg');
      expect(svg).not.toBeEmptyDOMElement();
    });

    it('should display ImageWarning with explicit props, Dark Mode', () => {
      const { view } = setupImageWarning({ darkLight: DarkLight.Dark });
      const svg = view.container.querySelector('svg');
      expect(screen.getByTitle('Warning')).toBeInTheDocument();
      expect(view.container.firstChild).toHaveClass(...['warning', 'dark']);
      expect(view.container.firstChild).toHaveAttribute('fill', 'none');
      expect(view.container.firstChild).toHaveAttribute('height', '100');
      expect(view.container.firstChild).toHaveAttribute('width', '100');
      expect(view.container.firstChild).toHaveAttribute('viewBox', '0 0 1024 1024');
      expect(view.container.firstChild).toHaveAttribute('xmlns', 'http://www.w3.org/2000/svg');
      expect(svg).not.toBeEmptyDOMElement();
    });
  });
});
