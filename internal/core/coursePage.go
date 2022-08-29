package core

import (
	"errors"
	"fmt"
	"log"
	"newJwCourseHelper/internal/dto"
	"strconv"
)

func (u *User) FindCourse() *User {
	if len(u.config.target) == 0 {
		u.e = errors.New("empty target, please set")
		return u
	}

	findClassBaseField := dto.MakeFindClassReq(u.getField()) // 获取基本参数
	for _, target := range u.getTarget() {
		findClassBaseField.FilterList = append(findClassBaseField.FilterList, target.Name) //获取目标课程号
	}
	//这里的eachLen是每个课程的课程号搜索后的个数，可以防止后续搜索出现冗余
	list, eachLen := u.getCourseList(findClassBaseField, u.info.special["firstKklxdmArr"], u.config.target) //通过用户传过来的参数得到待选列表，这里可以查询到不同大类的课程
	getClassDetailField := dto.MakeGetClassDetailReq(u.getField())
	tempSum := eachLen[0]
	j := 0
	for i := 0; i < len(list.TmpList); {
		if i == tempSum {
			j++
			tempSum += eachLen[j]
		}
		getClassDetailField.KchId = list.TmpList[tempSum-1].KchId //获取list中当前遍历元素的课程号
		for _, target := range u.getTarget() {
			getClassDetailField.FilterList = append(getClassDetailField.FilterList, target.Name) //获取目标课程号
		}
		details := u.getCourseDetail(getClassDetailField, u.info.special, u.config.target[j].Type) //获取课程详情
		if *details == nil {
			id := list.TmpList[tempSum-1].KchId
			for j := 0; j < len(list.TmpList); j++ {
				if list.TmpList[j].KchId == id {
					i++
				}
			}
			continue
		}
		var tempInt1, tempInt2 int
		for index, detail := range *details {
			for j := 0; j < len(list.TmpList); j++ {
				if list.TmpList[j].JxbId == detail.JxbId {
					list.TmpList[j].DetailList = &(*details)[index]
					tempInt1, _ = strconv.Atoi(list.TmpList[j].Yxzrs)
					tempInt2, _ = strconv.Atoi((*details)[index].Jxbrl)
					list.TmpList[j].HaveSet = tempInt1 < tempInt2
					i++
					break
				}
			}
		}
	}
	log.Printf("使用关键词 【 %s 】 共查询到 %d 门课程\n", u.getTarget(), len(list.TmpList))
	u.courses = list
	u.e = nil
	return u
}

// PrintFireCourseList 输出待选课的列表
func (u *User) PrintFireCourseList() *User {
	if u.Error() != nil {
		log.Printf("find an err: %v\n", u.Error())
		return u
	}
	if u.courses == nil {
		u.e = errors.New("empty course list, please use FindCourse first")
		log.Printf("find an err: %v\n", u.Error())
		return u
	}
	for i := 0; i < len(u.courses.TmpList) && u.courses.TmpList[i].DetailList != nil; i++ {
		fmt.Printf("【%02d】 %s 课程号 %s 班级号 %s    总容量 %s 已选 %s\n",
			i+1,
			u.courses.TmpList[i].Kcmc,
			u.courses.TmpList[i].Kch,
			u.courses.TmpList[i].Jxbmc,
			(*u.courses.TmpList[i].DetailList).Jxbrl,
			u.courses.TmpList[i].Yxzrs)
	}
	return u
}

func (u *User) FireCourses() ([]string, error) {
	if u.Error() != nil {
		log.Printf("find an err: %v", u.Error())
		return nil, u.Error()
	}
	if u.courses == nil {
		u.e = errors.New("empty course list, please use FindCourse first")
		log.Printf("find an err: %v", u.Error())
		return nil, u.Error()
	}

	fireList := u.courses.TmpList
	var success []string

	for i := 0; i < len(fireList) && fireList[i].DetailList != nil; i++ {
		// 跳过选课失败的课程 & 已选课程
		{
			if u.checkInErrList(fireList[i].Jxbmc) || u.checkChosen(fireList[i].Kch) {
				continue
			}
		}

		// 有余量则选课
		if fireList[i].HaveSet {

			prvChooseReq := dto.MakeChooseClassPrvReq(u.getField())
			prvChooseReq.JxbIds = (*fireList[i].DetailList).DoJxbId
			prvChooseReq.KchId = fireList[i].Kch
			prvChooseReq.Cxbj = fireList[i].Cxbj
			prvChooseReq.Xxkbj = fireList[i].Xxkbj

			err := u.prvChooseCourse(prvChooseReq)
			if err != nil {
				log.Printf("【err】 选择 %s 时发生错误： %v\n", fireList[i].Jxbmc, err.Error())
				u.config.errTag = append(u.config.errTag, fireList[i].Jxbmc)
				continue
			}
			success = append(success, fireList[i].Jxbmc)

			// 刷新已选课程
			c := u.getChosenCourse(dto.MakeGetChosenClassReq(u.getField()))
			if c == nil {
				u.e = errors.New("get user chosen course failed")
			} else {
				u.info.chosenCourse = c
			}
		}
	}
	return success, u.Error()
}

func (u *User) checkInErrList(m string) bool {
	for _, s := range u.config.errTag {
		if m == s {
			return true
		}
	}
	return false
}

func (u *User) checkChosen(m string) bool {
	for j := 0; j < len(*u.info.chosenCourse); j++ {
		if (*u.info.chosenCourse)[j].Kch == m {
			return true
		}
	}
	return false
}

func (u *User) PrintCourseChosenList() {
	if u.info.chosenCourse == nil || len(*u.info.chosenCourse) == 0 {
		u.e = errors.New("empty course list")
		log.Printf("find an err: %v\n", u.Error())
		return
	}
	cl := *u.info.chosenCourse
	for i := 0; i < len(cl); i++ {
		fmt.Printf("【%02d】 %s 课程号 %s 班级号 %s 教师 %s\n",
			i+1,
			cl[i].Kcmc,
			cl[i].Kch,
			cl[i].Jxbmc,
			cl[i].Sksj)
	}
}
