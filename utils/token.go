package utils

func CountTokens(text string) int {
	// 这里可以引入第三方库或自定义函数来计算 token 数量
	// 例如，使用 BPE 算法或其他分词算法
	// todo: 这里为了简单起见，暂时使用字符数作为 token 数量
	return len([]rune(text))
}

// TakeSentences 从句子列表中取出一段话，并返回剩余的句子列表
// lst 为句子列表, 由原始文本分割得到 (比如根据 \n 分隔)
// maxToken 为每段话的最大 token 数
func TakeSentences(lst []string, maxToken int) (paragraph []rune, left []string) {
	lenSentence := len(lst)
	if lenSentence == 0 {
		return nil, nil
	}
	if lenSentence == 1 {
		return []rune(lst[0]), nil
	}
	stash := lst[0]
	for i := 1; i < lenSentence; i++ {
		if len(lst[i]) == 0 {
			continue
		}
		sentence := lst[i]
		if CountTokens(stash)+CountTokens(sentence) > maxToken {
			return []rune(stash), lst[i:]
		}
		stash += "\n" + sentence
	}

	return []rune(stash), nil
}
