export const LANGUAGE_MAP: Record<number, { name: string; monacoLang: string }> = {
  71: { name: 'Python 3', monacoLang: 'python' },
  63: { name: 'JavaScript (Node.js)', monacoLang: 'javascript' },
  54: { name: 'C++ (GCC 9.2.0)', monacoLang: 'cpp' },
  62: { name: 'Java (OpenJDK 13.0.1)', monacoLang: 'java' },
  50: { name: 'C (GCC 9.2.0)', monacoLang: 'c' },
  60: { name: 'Go (1.13.5)', monacoLang: 'go' },
};

export const DEFAULT_CODE: Record<number, string> = {
  71: `# Write your solution here\ndef solution():\n    pass\n\nif __name__ == "__main__":\n    solution()\n`,
  63: `// Write your solution here\nfunction solution() {\n    return null;\n}\n\nsolution();\n`,
  54: `#include <iostream>\nusing namespace std;\n\nint main() {\n    // Write your C++ solution here\n    return 0;\n}\n`,
  62: `import java.util.*;\n    public class Main {\n    public static void main(String[] args) {\n        // Write your Java solution here\n    }\n}\n`,
  50: `#include <stdio.h>\n\nint main() {\n    // Write your C solution here\n    return 0;\n}\n`,
  60: `package main\n\nimport "fmt"\n\nfunc main() {\n    // Write your Go solution here\n}\n`,
};
