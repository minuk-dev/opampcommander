# OpAMP Commander Documentation

이 디렉토리는 OpAMP Commander의 공식 문서를 포함합니다.

## 로컬 개발

문서 사이트를 로컬에서 실행하려면:

### 사전 요구사항

- [Hugo Extended](https://gohugo.io/installation/) v0.110.0 이상
- Node.js 및 npm (선택사항, PostCSS 처리용)

### 실행

```bash
# 개발 서버 시작
hugo server -D

# 또는 npm을 사용하여
npm install
npm run dev
```

문서 사이트는 `http://localhost:1313`에서 확인할 수 있습니다.

## 빌드

프로덕션 빌드를 생성하려면:

```bash
hugo --minify
```

빌드된 파일은 `public/` 디렉토리에 생성됩니다.

## 문서 작성

새로운 문서 페이지를 추가하려면 `content/ko/docs/` 디렉토리에 마크다운 파일을 생성하세요.

```bash
hugo new content/ko/docs/your-section/your-page.md
```

## 테마

이 문서는 [Docsy](https://www.docsy.dev/) 테마를 사용합니다.

## 기여

문서 개선에 기여해주세요! Pull Request를 환영합니다.
