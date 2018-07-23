
// 		inputDir := filepath.Join("job", j.ID.String(), "input")
// 		if err = os.MkdirAll(inputDir, 0755); err != nil {
// 			app.Sugar.Errorw("failed to create job directory",
// 				"url", r.URL,
// 				"err", err.Error(),
// 				"jID", j.ID,
// 			)
// 			_ = db.SetJobInactive(r, j.ID)
// 			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
// 		}
//
// 		vals := []string{"requirements", "main", "data"}
// 		for i := range vals {
// 			err = uploadAndCacheFormFile(r, inputDir, vals[i])
// 			if err != nil {
// 				app.Sugar.Errorw("failed to upload form file",
// 					"url", r.URL,
// 					"err", err.Error(),
// 					"jID", j.ID,
// 				)
// 				_ = db.SetJobInactive(r, j.ID)
// 				return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
// 			}
// 		}
//
// 		sqlStmt = `
// 	UPDATE statuses
// 	SET (user_data_stored) = ($1)
// 	WHERE job_uuid = $2
// 	`
// 		if _, err = db.Db.Exec(sqlStmt, true, j.ID); err != nil {
// 			pqErr := err.(*pq.Error)
// 			if pqErr.Fatal() {
// 				app.Sugar.Fatalw("failed to insert status",
// 					"url", r.URL,
// 					"err", err.Error(),
// 					"jID", j.ID,
// 					"pq_sev", pqErr.Severity,
// 					"pq_code", pqErr.Code,
// 					"pq_detail", pqErr.Detail,
// 				)
// 			} else {
// 				app.Sugar.Errorw("failed to insert status",
// 					"url", r.URL,
// 					"err", err.Error(),
// 					"jID", j.ID,
// 					"pq_sev", pqErr.Severity,
// 					"pq_code", pqErr.Code,
// 					"pq_detail", pqErr.Detail,
// 				)
// 			}
// 			_ = db.SetJobInactive(r, j.ID)
// 			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
// 		}
//
// 		return nil
// 	}
//
// func uploadAndCacheFormFile(r *http.Request, dir, val string) error {
// 	f, _, err := r.FormFile(val)
// 	if err != nil {
// 		return err
// 	}
// 	defer app.CheckErr(r, f.Close)
//
// 	p := filepath.Join(dir, val)
// 	file, err := os.Create(p)
// 	if err != nil {
// 		return err
// 	}
// 	defer app.CheckErr(r, file.Close)
// 	tee := io.TeeReader(f, file)
//
// 	ctx := r.Context()
// 	ow := bkt.Object(p).NewWriter(ctx)
// 	_, err = io.Copy(ow, tee)
// 	_, err = io.Copy(ow, f)
// 	if err != nil {
// 		return err
// 	}
// 	if err = ow.Close(); err != nil {
// 		return err
// 	}
//
// 	return nil
// }
