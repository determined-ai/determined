import { Router as RemixRouter } from '@remix-run/router';
import React from 'react';

import router from 'router';
import { AnyMouseEvent, reactHostAddress } from 'utils/routes';
import * as routes from 'utils/routes';

const setup = (url = '/', base = 'http://www.example.com') => {
  const newUrl = new URL(url, base);
  Object.defineProperty(window, 'location', {
    value: {
      hash: newUrl.hash,
      host: newUrl.host,
      hostname: newUrl.hostname,
      href: newUrl.href,
      origin: newUrl.origin,
      password: newUrl.password,
      pathname: newUrl.pathname.toString(),
      port: newUrl.port,
      protocol: newUrl.protocol,
      search: newUrl.search,
      username: newUrl.username,
    },
    writable: true,
  });
  const location: Location = window.location;
  return { location };
};

beforeEach(() => {
  setup();
});

describe('Routes Utilities', () => {
  describe('isFullPath', () => {
    it('should validate full path', () => {
      expect(routes.isFullPath('https://determinedai.atlassian.net')).toBeTruthy();
      expect(routes.isFullPath('http://determinedai.atlassian.net')).toBeTruthy();
      expect(routes.isFullPath('http://localhost:3000/det')).toBeTruthy();
      expect(routes.isFullPath('https://localhost:3000/det')).toBeTruthy();
      expect(routes.isFullPath('https://localhost:3000/')).toBeTruthy();
      expect(routes.isFullPath('https://localhost:3000')).toBeTruthy();
    });

    it('should invalidate full path', () => {
      expect(routes.isFullPath('')).toBeFalsy();
      expect(routes.isFullPath('determinedai.atlassian.net/browse/DET-7902')).toBeFalsy();
      expect(routes.isFullPath('_https://localhost:3000/det')).toBeFalsy();
      expect(routes.isFullPath('abs_https://localhost:3000/det')).toBeFalsy();
      expect(routes.isFullPath('ahttp')).toBeFalsy();
      expect(routes.isFullPath('git+ssh://example.con/item')).toBeFalsy();
      expect(routes.isFullPath('https')).toBeFalsy();
      expect(routes.isFullPath('http')).toBeFalsy();
    });
  });

  describe('isAbsolutePath', () => {
    it('should validate absolute path', () => {
      expect(routes.isAbsolutePath('/')).toBeTruthy();
      expect(routes.isAbsolutePath('/det')).toBeTruthy();
      expect(routes.isAbsolutePath('/det/projects')).toBeTruthy();
      expect(routes.isAbsolutePath('/det/projects/')).toBeTruthy();
    });

    it('should invalidate absolute path', () => {
      expect(routes.isAbsolutePath('')).toBeFalsy();
      expect(routes.isAbsolutePath('https://localhost:3000/det')).toBeFalsy();
      expect(routes.isAbsolutePath('git+ssh://example.con/item')).toBeFalsy();
      expect(routes.isAbsolutePath('asdf/')).toBeFalsy();
      expect(routes.isAbsolutePath('a')).toBeFalsy();
      expect(routes.isAbsolutePath('///')).toBeFalsy();
      expect(routes.isAbsolutePath('//')).toBeFalsy();
      expect(routes.isAbsolutePath('/asdf//')).toBeFalsy();
      expect(routes.isAbsolutePath('//det/projects')).toBeFalsy();
    });
  });

  describe('locationToPath', () => {
    it('should return null with default url', () => {
      const nullableLocation: Location | undefined = undefined;
      expect(routes.locationToPath(nullableLocation)).toBeNull();
    });

    it('should return string with simple url', () => {
      const { location } = setup();
      expect(routes.locationToPath(location)).toBe('/');
    });

    it('should return string with complex url without query', () => {
      const { location } = setup('/mock/test');
      expect(routes.locationToPath(location)).toBe('/mock/test');
    });

    it('should return string with complex url with query', () => {
      const { location } = setup('/mock/test?query=true');
      expect(routes.locationToPath(location)).toBe('/mock/test?query=true');
    });

    it('should return string with complex url with hash', () => {
      const { location } = setup('/mock/test#hash');
      expect(routes.locationToPath(location)).toBe('/mock/test#hash');
    });

    it('should return string with complex url with query and hash', () => {
      const { location } = setup('/mock/test?query=true&id=3#hash');
      expect(routes.locationToPath(location)).toBe('/mock/test?query=true&id=3#hash');
    });
  });

  describe('windowOpenFeatures', () => {
    it('should contain items', () => {
      const items = routes.windowOpenFeatures;
      expect(items).toHaveLength(2);
      expect(items).toContain('noopener');
      expect(items).toContain('noreferrer');
    });
  });

  describe('openBlank', () => {
    let originalWindowOpen: typeof window.open;

    beforeEach(() => {
      originalWindowOpen = window.open;
      window.open = vi.fn();
    });

    afterEach(() => {
      window.open = originalWindowOpen;
    });

    it('should direct to https://localhost:3000', () => {
      expect(window.open).not.toHaveBeenCalled();
      routes.openBlank('https://localhost:3000');
      expect(window.open).toHaveBeenCalledTimes(1);
      expect(window.open).toHaveBeenCalledWith(
        'https://localhost:3000',
        '_blank',
        'noopener,noreferrer',
      );
    });

    it('should direct to https://localhost:3000/det/projects/1?test=true', () => {
      expect(window.open).not.toHaveBeenCalled();
      routes.openBlank('https://localhost:3000/det/projects/1?test=true');
      expect(window.open).toHaveBeenCalledTimes(1);
      expect(window.open).toHaveBeenCalledWith(
        'https://localhost:3000/det/projects/1?test=true',
        '_blank',
        'noopener,noreferrer',
      );
    });

    it('should direct to /test', () => {
      expect(window.open).not.toHaveBeenCalled();
      routes.openBlank('/test');
      expect(window.open).toHaveBeenCalledTimes(1);
      expect(window.open).toHaveBeenCalledWith('/test', '_blank', 'noopener,noreferrer');
    });
  });

  describe('isMouseEvent', () => {
    it('should be MouseEvent', () => {
      const e: AnyMouseEvent = new MouseEvent('click', { bubbles: true });
      expect(routes.isMouseEvent(e)).toBeTruthy();
    });

    it('should not be MouseEvent', () => {
      const e = new KeyboardEvent('keydown', {
        bubbles: true,
        cancelable: true,
        key: 'q',
        keyCode: 81,
        shiftKey: true,
      }) as unknown as React.KeyboardEvent;
      expect(routes.isMouseEvent(e)).toBeFalsy();
    });
  });

  describe('isNewTabClickEvent', () => {
    it('should be NewTabClickEvent', () => {
      const e1: AnyMouseEvent = new MouseEvent('click', {
        button: 1,
        ctrlKey: false,
        metaKey: false,
      });
      expect(routes.isNewTabClickEvent(e1)).toBeTruthy();

      const e2: AnyMouseEvent = new MouseEvent('click', {
        button: 0,
        ctrlKey: false,
        metaKey: true,
      });
      expect(routes.isNewTabClickEvent(e2)).toBeTruthy();

      const e3: AnyMouseEvent = new MouseEvent('click', {
        button: 0,
        ctrlKey: true,
        metaKey: false,
      });
      expect(routes.isNewTabClickEvent(e3)).toBeTruthy();
    });

    it('should not be NewTabClickEvent', () => {
      const e: AnyMouseEvent = new MouseEvent('click', {
        button: 0,
        ctrlKey: false,
        metaKey: false,
      });
      expect(routes.isNewTabClickEvent(e)).toBeFalsy();
    });
  });

  describe('reactHostAddress', () => {
    it('should invoke reactHostAddress with example.com', () => {
      setup('/', 'http://www.example.com');
      expect(reactHostAddress()).toBe('http://www.example.com');
    });

    it('should invoke reactHostAddress with determined.com', () => {
      setup('/', 'https://www.determined.ai/');
      expect(reactHostAddress()).toBe('https://www.determined.ai');
    });

    it('should invoke reactHostAddress with determined.ai/project', () => {
      setup('/', 'https://www.determined.ai/project');
      expect(reactHostAddress()).toBe('https://www.determined.ai');
    });
  });

  describe('ensureAbsolutePath', () => {
    it('should ensure absolute path', () => {
      expect(routes.ensureAbsolutePath('')).toBe('/');
      expect(routes.ensureAbsolutePath('/')).toBe('/');
      expect(routes.ensureAbsolutePath('test')).toBe('/test');
      expect(routes.ensureAbsolutePath('/test')).toBe('/test');
      expect(routes.ensureAbsolutePath('/test/nested')).toBe('/test/nested');
      expect(routes.ensureAbsolutePath('test/nested')).toBe('/test/nested');
    });
  });

  describe('filterOutLoginLocation', () => {
    it('should filter out login location', () => {
      const { location } = setup('/login', 'http://localhost:3000');
      const loc = { pathname: location.pathname };
      loc.pathname = location.pathname;
      expect(routes.filterOutLoginLocation(loc)).toBeUndefined();
    });

    it('should filter out login location when nested pathname has login', () => {
      const { location } = setup('test/test/login', 'http://localhost:3000');
      const loc = { pathname: location.pathname };
      loc.pathname = location.pathname;
      expect(routes.filterOutLoginLocation(loc)).toBeUndefined();
    });

    it('should get non-Login location', () => {
      const { location } = setup('test', 'http://localhost:3000');
      const loc = { pathname: location.pathname };
      loc.pathname = location.pathname;
      expect(routes.filterOutLoginLocation(loc)).toMatchObject({ pathname: '/test' });
    });

    it('should get non-Login location with nested pathname', () => {
      const { location } = setup('test/project', 'http://localhost:3000');
      const loc = { pathname: location.pathname };
      loc.pathname = location.pathname;
      expect(routes.filterOutLoginLocation(loc)).toMatchObject({ pathname: '/test/project' });
    });
  });

  describe('parseUrl', () => {
    it('should parse URL with base url', () => {
      const url = 'https://www.determined.ai/';
      setup('/', url);
      expect(routes.parseUrl(url)).toMatchObject(new URL(url));
    });

    it('should parse URL with pathname and query', () => {
      const url = 'https://www.determined.ai/projects?testid=100';
      setup('/', url);
      expect(routes.parseUrl(url)).toMatchObject(new URL(url));
    });

    it('should parse URL without base', () => {
      const base = 'https://www.determined.ai';
      const url = '/projects?testid=100';
      setup(url, base);
      expect(routes.parseUrl(url)).toMatchObject(new URL(`${base}${url}`));
    });
  });

  describe('routeToExternalUrl', () => {
    beforeEach(() => {
      window.location.assign = vi.fn();
    });

    it('should route to external URL', () => {
      const externalUrl = 'https://www.determined.ai/blog';
      expect(window.location.assign).not.toHaveBeenCalled();
      routes.routeToExternalUrl(externalUrl);
      expect(window.location.assign).toHaveBeenCalledTimes(1);
      expect(window.location.assign).toHaveBeenCalledWith(externalUrl);
    });

    it('should route to external URL ver2', () => {
      const externalUrl = 'https://www.apple.com';
      expect(window.location.assign).not.toHaveBeenCalled();
      routes.routeToExternalUrl(externalUrl);
      expect(window.location.assign).toHaveBeenCalledTimes(1);
      expect(window.location.assign).toHaveBeenCalledWith(externalUrl);
    });
  });

  describe('routeToReactUrl', () => {
    let instance: RemixRouter;

    beforeEach(() => {
      instance = router.getRouter();
      vi.spyOn<RemixRouter, 'navigate'>(instance, 'navigate');
    });

    afterEach(() => {
      vi.clearAllMocks();
    });

    it('should route to react URL', () => {
      const path = '/clusters';
      expect(instance.navigate).not.toHaveBeenCalled();
      routes.routeToReactUrl(path);
      expect(instance.navigate).toHaveBeenCalledTimes(1);
      expect(instance.navigate).toHaveBeenCalledWith(path);
    });

    it('should route to react URL with determined.ai base url', () => {
      setup('/', 'https://www.determined.ai');
      const path = '/dashboard';
      expect(instance.navigate).not.toHaveBeenCalled();
      routes.routeToReactUrl(path);
      expect(instance.navigate).toHaveBeenCalledTimes(1);
      expect(instance.navigate).toHaveBeenCalledWith(path);
    });
  });
});
