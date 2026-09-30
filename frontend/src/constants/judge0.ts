/**
 * Judge0 API Constants
 * Reference: https://judge0.com/documentation/
 */

export interface Judge0StatusConfig {
  id: number;
  label: string;
  shortLabel: string;
  color: string;
  bgColor: string;
  icon: string;
}

export interface LanguageConfig {
  id: number;
  name: string;
  monacoLang: string;
  extension: string;
  defaultCode: string;
}

export interface DifficultyConfig {
  value: string;
  label: string;
  color: string;
  class: string;
}

/**
 * Judge0 Status IDs and their display configurations
 */
export const JUDGE0_STATUS: Record<string, Judge0StatusConfig> = {
  IN_QUEUE: {
    id: 1,
    label: 'In Queue',
    shortLabel: 'Queued',
    color: '#f59e0b',
    bgColor: '#f59e0b15',
    icon: 'clock',
  },
  PROCESSING: {
    id: 2,
    label: 'Processing',
    shortLabel: 'Running',
    color: '#f59e0b',
    bgColor: '#f59e0b15',
    icon: 'loader',
  },
  ACCEPTED: {
    id: 3,
    label: 'Accepted',
    shortLabel: 'AC',
    color: '#10b981',
    bgColor: '#10b98115',
    icon: 'check',
  },
  WRONG_ANSWER: {
    id: 4,
    label: 'Wrong Answer',
    shortLabel: 'WA',
    color: '#ef4444',
    bgColor: '#ef444415',
    icon: 'x',
  },
  TIME_LIMIT_EXCEEDED: {
    id: 5,
    label: 'Time Limit Exceeded',
    shortLabel: 'TLE',
    color: '#f59e0b',
    bgColor: '#f59e0b15',
    icon: 'clock',
  },
  COMPILATION_ERROR: {
    id: 6,
    label: 'Compilation Error',
    shortLabel: 'CE',
    color: '#ef4444',
    bgColor: '#ef444415',
    icon: 'alert',
  },
  RUNTIME_ERROR_SIGSEGV: {
    id: 7,
    label: 'Runtime Error (SIGSEGV)',
    shortLabel: 'RTE',
    color: '#ef4444',
    bgColor: '#ef444415',
    icon: 'alert',
  },
  RUNTIME_ERROR_SIGXFSZ: {
    id: 8,
    label: 'Runtime Error (SIGXFSZ)',
    shortLabel: 'RTE',
    color: '#ef4444',
    bgColor: '#ef444415',
    icon: 'alert',
  },
  RUNTIME_ERROR_SIGFPE: {
    id: 9,
    label: 'Runtime Error (SIGFPE)',
    shortLabel: 'RTE',
    color: '#ef4444',
    bgColor: '#ef444415',
    icon: 'alert',
  },
  RUNTIME_ERROR_SIGABRT: {
    id: 10,
    label: 'Runtime Error (SIGABRT)',
    shortLabel: 'RTE',
    color: '#ef4444',
    bgColor: '#ef444415',
    icon: 'alert',
  },
  RUNTIME_ERROR_NZEC: {
    id: 11,
    label: 'Runtime Error (NZEC)',
    shortLabel: 'RTE',
    color: '#ef4444',
    bgColor: '#ef444415',
    icon: 'alert',
  },
  RUNTIME_ERROR_OTHER: {
    id: 12,
    label: 'Runtime Error (Other)',
    shortLabel: 'RTE',
    color: '#ef4444',
    bgColor: '#ef444415',
    icon: 'alert',
  },
  INTERNAL_ERROR: {
    id: 13,
    label: 'Internal Error',
    shortLabel: 'IE',
    color: '#ef4444',
    bgColor: '#ef444415',
    icon: 'alert',
  },
  EXEC_LIMIT_EXCEEDED: {
    id: 14,
    label: 'Exec Limit Exceeded',
    shortLabel: 'ELE',
    color: '#f59e0b',
    bgColor: '#f59e0b15',
    icon: 'clock',
  },
};

/**
 * Get status info by ID
 */
export const getStatusInfo = (statusId: number): Judge0StatusConfig => {
  const status = Object.values(JUDGE0_STATUS).find((s) => s.id === statusId);
  return (
    status || {
      id: statusId,
      label: 'Unknown',
      shortLabel: '?',
      color: '#6b7280',
      bgColor: '#6b728015',
      icon: 'question',
    }
  );
};

/**
 * Check if status is a passing/accepted status
 */
export const isAcceptedStatus = (statusId: number): boolean => {
  return statusId === JUDGE0_STATUS.ACCEPTED.id;
};

/**
 * Check if status is an error status
 */
export const isErrorStatus = (statusId: number): boolean => {
  return [
    JUDGE0_STATUS.COMPILATION_ERROR.id,
    JUDGE0_STATUS.RUNTIME_ERROR_SIGSEGV.id,
    JUDGE0_STATUS.RUNTIME_ERROR_SIGXFSZ.id,
    JUDGE0_STATUS.RUNTIME_ERROR_SIGFPE.id,
    JUDGE0_STATUS.RUNTIME_ERROR_SIGABRT.id,
    JUDGE0_STATUS.RUNTIME_ERROR_NZEC.id,
    JUDGE0_STATUS.RUNTIME_ERROR_OTHER.id,
    JUDGE0_STATUS.INTERNAL_ERROR.id,
  ].includes(statusId);
};

/**
 * Check if status is a timeout status
 */
export const isTimeoutStatus = (statusId: number): boolean => {
  return [JUDGE0_STATUS.TIME_LIMIT_EXCEEDED.id, JUDGE0_STATUS.EXEC_LIMIT_EXCEEDED.id].includes(
    statusId
  );
};

/**
 * Language configurations
 */
export const LANGUAGE_CONFIG: Record<number, LanguageConfig> = {
  71: {
    id: 71,
    name: 'Python 3',
    monacoLang: 'python',
    extension: 'py',
    defaultCode: `# Common imports for competitive programming
import sys
from typing import List, Tuple, Dict, Set

def solution() -> None:
    """Write your solution here."""
    # Read input
    # data = sys.stdin.read().strip().split()
    # Process and print output
    pass

if __name__ == "__main__":
    solution()
`,
  },
  63: {
    id: 63,
    name: 'JavaScript (Node.js)',
    monacoLang: 'javascript',
    extension: 'js',
    defaultCode: `// Common utilities for competitive programming
const readline = require('readline');

const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout
});

function solution(input) {
    // Write your solution here
    // Process input and return output
    return input;
}

// Read all input
let input = [];
rl.on('line', (line) => {
    input.push(line);
}).on('close', () => {
    const result = solution(input);
    if (result) console.log(result);
});
`,
  },
  54: {
    id: 54,
    name: 'C++ (GCC 9.2.0)',
    monacoLang: 'cpp',
    extension: 'cpp',
    defaultCode: `#include <bits/stdc++.h>
using namespace std;

#define int long long
#define all(x) (x).begin(), (x).end()
#define rall(x) (x).rbegin(), (x).rend()

void solution() {
    // Write your C++ solution here
    int n;
    cin >> n;
    // Process and print output
}

int32_t main() {
    ios_base::sync_with_stdio(false);
    cin.tie(NULL);
    solution();
    return 0;
}
`,
  },
  62: {
    id: 62,
    name: 'Java (OpenJDK 13.0.1)',
    monacoLang: 'java',
    extension: 'java',
    defaultCode: `import java.io.*;
import java.util.*;

public class Main {
    static BufferedReader br = new BufferedReader(new InputStreamReader(System.in));
    static StringTokenizer st;
    static PrintWriter out = new PrintWriter(new BufferedOutputStream(System.out));

    static String next() throws IOException {
        while (st == null || !st.hasMoreTokens()) {
            st = new StringTokenizer(br.readLine());
        }
        return st.nextToken();
    }

    static int nextInt() throws IOException {
        return Integer.parseInt(next());
    }

    static long nextLong() throws IOException {
        return Long.parseLong(next());
    }

    static void solution() throws IOException {
        // Write your Java solution here
        int n = nextInt();
        // Process and print output
        out.println(n);
    }

    public static void main(String[] args) throws IOException {
        solution();
        out.flush();
        out.close();
    }
}
`,
  },
  50: {
    id: 50,
    name: 'C (GCC 9.2.0)',
    monacoLang: 'c',
    extension: 'c',
    defaultCode: `#include <stdio.h>
#include <stdlib.h>

void solution() {
    // Write your C solution here
    int n;
    scanf("%d", &n);
    // Process and print output
    printf("%d\\n", n);
}

int main() {
    solution();
    return 0;
}
`,
  },
  60: {
    id: 60,
    name: 'Go (1.13.5)',
    monacoLang: 'go',
    extension: 'go',
    defaultCode: `package main

import (
    "bufio"
    "fmt"
    "os"
    "strconv"
    "strings"
)

func solution() {
    // Write your Go solution here
    scanner := bufio.NewScanner(os.Stdin)
    scanner.Scan()
    n, _ := strconv.Atoi(scanner.Text())
    // Process and print output
    fmt.Println(n)
}

func main() {
    solution()
}
`,
  },
};

/**
 * Get language config by ID
 */
export const getLanguageConfig = (languageId: number): LanguageConfig | null => {
  return LANGUAGE_CONFIG[languageId] || null;
};

/**
 * Get all available languages
 */
export const getAllLanguages = (): LanguageConfig[] => {
  return Object.values(LANGUAGE_CONFIG);
};

/**
 * Difficulty levels
 */
export const DIFFICULTY: Record<string, DifficultyConfig> = {
  EASY: {
    value: 'easy',
    label: 'Easy',
    color: '#10b981',
    class: 'difficulty-easy',
  },
  MEDIUM: {
    value: 'medium',
    label: 'Medium',
    color: '#f59e0b',
    class: 'difficulty-medium',
  },
  HARD: {
    value: 'hard',
    label: 'Hard',
    color: '#ef4444',
    class: 'difficulty-hard',
  },
};

/**
 * Get difficulty config by value
 */
export const getDifficultyConfig = (difficulty: string): DifficultyConfig => {
  const diff = Object.values(DIFFICULTY).find((d) => d.value === difficulty);
  return diff || DIFFICULTY.MEDIUM;
};

/**
 * Default code templates by language ID
 */
export const DEFAULT_CODE: Record<number, string> = Object.fromEntries(
  Object.values(LANGUAGE_CONFIG).map((lang) => [lang.id, lang.defaultCode])
);

/**
 * Local storage keys
 */
export const STORAGE_KEYS = {
  PROBLEM_CODE: (problemId: number | string, userRegdNo: string): string =>
    `problem_code_${problemId}_${userRegdNo}`,
  SOLVE_SESSION: (problemId: number | string, userRegdNo: string): string =>
    `solve_session_${problemId}_${userRegdNo}`,
  DRAFT_CODE: (problemId: number | string, userRegdNo: string): string =>
    `draft_code_${problemId}_${userRegdNo}`,
};

/**
 * Time formatting utilities
 */
export const formatTime = (seconds: number): string => {
  const hrs = Math.floor(seconds / 3600);
  const mins = Math.floor((seconds % 3600) / 60);
  const secs = seconds % 60;

  if (hrs > 0) {
    return `${hrs}:${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
  }
  return `${mins}:${secs.toString().padStart(2, '0')}`;
};

export const formatTimeAgo = (date: string | Date): string => {
  const seconds = Math.floor((new Date().getTime() - new Date(date).getTime()) / 1000);

  if (seconds < 60) return 'just now';
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
  if (seconds < 604800) return `${Math.floor(seconds / 86400)}d ago`;
  return new Date(date).toLocaleDateString();
};

/**
 * Memory formatting
 */
export const formatMemory = (bytesKB: number): string => {
  const mb = bytesKB / 1024;
  if (mb >= 1024) {
    return `${(mb / 1024).toFixed(2)} GB`;
  }
  return `${mb.toFixed(2)} MB`;
};
