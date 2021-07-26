import { ansiToHtml } from './dom';

describe('ansiToHtml', () => {
  it('should convert ANSI colors', () => {
    expect(ansiToHtml('\u001b[30m30\u001b[0m')).toBe('<span style="color:#000">30</span>');
    expect(ansiToHtml('\u001b[31m31\u001b[0m')).toBe('<span style="color:#A00">31</span>');
    expect(ansiToHtml('\u001b[32m32\u001b[0m')).toBe('<span style="color:#0A0">32</span>');
    expect(ansiToHtml('\u001b[33m33\u001b[0m')).toBe('<span style="color:#A50">33</span>');
    expect(ansiToHtml('\u001b[34m34\u001b[0m')).toBe('<span style="color:#00A">34</span>');
    expect(ansiToHtml('\u001b[35m35\u001b[0m')).toBe('<span style="color:#A0A">35</span>');
    expect(ansiToHtml('\u001b[36m36\u001b[0m')).toBe('<span style="color:#0AA">36</span>');
  });

  it('should escape harmful unicode characters', () => {
    expect(ansiToHtml('\u003c\u003e\u0022\u0026\u0027')).toBe('&lt;&gt;&quot;&amp;&apos;');
  });
});
