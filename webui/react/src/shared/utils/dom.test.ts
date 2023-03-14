import * as utils from './dom';

const mockNavigatorClipboard = () => {
  Object.defineProperty(navigator, 'clipboard', {
    configurable: true,
    value: {
      readText: jest.fn(),
      writeText: jest.fn(),
    },
    writable: true,
  });
};

const unmockNavigatorClipboard = () => {
  Object.defineProperty(navigator, 'clipboard', { value: undefined });
};

describe('DOM Utilities', () => {
  describe('ansiToHtml', () => {
    it('should convert ANSI colors', () => {
      expect(utils.ansiToHtml('\u001b[30m30\u001b[0m')).toBe('<span style="color:#000">30</span>');
      expect(utils.ansiToHtml('\u001b[31m31\u001b[0m')).toBe('<span style="color:#A00">31</span>');
      expect(utils.ansiToHtml('\u001b[32m32\u001b[0m')).toBe('<span style="color:#0A0">32</span>');
      expect(utils.ansiToHtml('\u001b[33m33\u001b[0m')).toBe('<span style="color:#A50">33</span>');
      expect(utils.ansiToHtml('\u001b[34m34\u001b[0m')).toBe('<span style="color:#00A">34</span>');
      expect(utils.ansiToHtml('\u001b[35m35\u001b[0m')).toBe('<span style="color:#A0A">35</span>');
      expect(utils.ansiToHtml('\u001b[36m36\u001b[0m')).toBe('<span style="color:#0AA">36</span>');
    });

    it('should escape harmful unicode characters', () => {
      expect(utils.ansiToHtml('\u003c\u003e\u0022\u0026\u0027')).toBe('&lt;&gt;&quot;&amp;&apos;');
    });
  });

  describe('copyToClipboard and readFromClipboard', () => {
    it('should error writing to clipboard when navigator.clipboard not available', async () => {
      await expect(utils.copyToClipboard('Hello World')).rejects.toBeInstanceOf(Error);
    });

    it('should error reading from clipboard when navigator.clipboard not available', async () => {
      await expect(utils.readFromClipboard()).rejects.toBeInstanceOf(Error);
    });

    // Not the ideal way to test but `navigator.clipboard` is not available in headless mode.
    it('should be able to read and write to clipboard', () => {
      mockNavigatorClipboard();

      const content = 'Copy this!';
      utils.copyToClipboard(content);
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith(content);

      utils.readFromClipboard();
      expect(navigator.clipboard.readText).toHaveBeenCalled();

      unmockNavigatorClipboard();
    });
  });
});
