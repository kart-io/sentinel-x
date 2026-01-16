import subprocess
import sys

def get_git_diff():
    # 获取已暂存的更改
    try:
        result = subprocess.run(
            ['git', 'diff', '--cached', '--stat'],
            capture_output=True, text=True, check=True
        )
        return result.stdout
    except subprocess.CalledProcessError:
        return None

def main():
    diff = get_git_diff()
    if not diff:
        print("错误：没有检测到已暂存（staged）的更改。请先运行 `git add`。")
        sys.exit(1)

    print("已检测到以下更改，请根据此内容生成 Conventional Commit 信息：")
    print("-" * 30)
    print(diff)
    print("-" * 30)
    print("\n建议格式：<type>(<scope>): <subject>")

if __name__ == "__main__":
    main()